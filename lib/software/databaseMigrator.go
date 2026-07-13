package software

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
)

// databaseServerPackages are relational-database server packages whose actual
// payload lives in a data directory (e.g. /var/lib/mysql), not in files apt
// installs. Installing the package on the target creates an empty server, so the
// databases, tables and users must be migrated separately or dependent apps (e.g.
// WordPress) fail with "Error establishing a database connection".
var databaseServerPackages = map[string]bool{
	"mariadb-server":        true,
	"mysql-server":          true,
	"percona-server-server": true,
	"default-mysql-server":  true,
}

func isDatabaseServerPackage(name string) bool {
	if databaseServerPackages[name] {
		return true
	}
	// Versioned variants such as mysql-server-8.0 / mariadb-server-10.6.
	for _, p := range []string{"mysql-server-", "mariadb-server-", "percona-server-server-"} {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// dbDumpScript dumps every non-system database (schema + data) and every
// application user (with its password hash and grants) into two files under /tmp.
// It is delivered base64-encoded so no shell/SQL quoting is mangled in transit.
// %[1]s is the dump path, %[2]s the users path.
const dbDumpScript = `set -e
DUMP="%[1]s"
USERS="%[2]s"
DBS=$(mysql -N -e "SHOW DATABASES" | grep -vE "^(information_schema|performance_schema|mysql|sys)$" | tr '\n' ' ')
if [ -z "$DBS" ]; then
  echo "GRASSHOPPER_NO_USER_DB"
  exit 0
fi
mysqldump --single-transaction --quick --routines --events --databases $DBS > "$DUMP"
: > "$USERS"
mysql -N -e "SELECT CONCAT('''', user, '''@''', host, '''') FROM mysql.user WHERE user NOT IN ('root','mysql','mariadb.sys','debian-sys-maint','mysql.session','mysql.sys','mysql.infoschema') AND user <> ''" | while read acc; do
  [ -z "$acc" ] && continue
  mysql -N -e "SHOW CREATE USER $acc" | sed 's/$/;/' >> "$USERS"
  mysql -N -e "SHOW GRANTS FOR $acc" | sed 's/$/;/' >> "$USERS"
done
echo "GRASSHOPPER_DB_DUMPED $DBS"
`

// dbImportScript imports the dump and users produced by dbDumpScript on the
// target and reloads privileges. %[1]s is the dump path, %[2]s the users path.
const dbImportScript = `set -e
DUMP="%[1]s"
USERS="%[2]s"
if [ ! -f "$DUMP" ]; then
  echo "GRASSHOPPER_NO_DUMP"
  exit 0
fi
mysql < "$DUMP"
if [ -f "$USERS" ]; then
  mysql < "$USERS" || true
fi
mysql -e "FLUSH PRIVILEGES;"
echo "GRASSHOPPER_DB_IMPORTED"
`

// runEncodedScript delivers script to the host base64-encoded and runs it with
// sh, avoiding any quoting interaction with the sudo wrapper.
func runEncodedScript(client *ssh.Client, script string) (string, error) {
	encoded := base64.StdEncoding.EncodeToString([]byte(script))
	cmd := fmt.Sprintf("echo %s | base64 -d | sh", encoded)
	return runSSHCommand(client, cmd)
}

// migrateSQLDatabase migrates the databases and users of a MySQL/MariaDB server
// from the source to the target using a logical dump (safe on a live server,
// unlike copying the raw data directory). It is a no-op when the server has no
// user databases. Non-fatal: the package install itself already succeeded, so a
// dump/import problem is logged for the operator rather than failing the package.
func migrateSQLDatabase(sourceClient, targetClient *ssh.Client, uuid string, migrationLogger *Logger) error {
	dumpPath := fmt.Sprintf("/tmp/grasshopper_dbdump_%s.sql", uuid)
	usersPath := fmt.Sprintf("/tmp/grasshopper_dbusers_%s.sql", uuid)

	migrationLogger.Printf(INFO, "Dumping databases on source for import to target\n")
	out, err := runEncodedScript(sourceClient, fmt.Sprintf(dbDumpScript, dumpPath, usersPath))
	if err != nil {
		return fmt.Errorf("failed to dump source databases: %s", strings.TrimSpace(out))
	}
	if strings.Contains(out, "GRASSHOPPER_NO_USER_DB") {
		migrationLogger.Printf(INFO, "No user databases to migrate\n")
		return nil
	}
	migrationLogger.Printf(INFO, "Source databases dumped: %s\n", strings.TrimSpace(out))

	// Transfer the dump (and users) to the same path on the target.
	if err := copyPathWithChunks(sourceClient, targetClient, dumpPath, uuid, migrationLogger); err != nil {
		return fmt.Errorf("failed to transfer database dump: %v", err)
	}
	if remotePathType(sourceClient, usersPath) == "file" {
		if err := copyPathWithChunks(sourceClient, targetClient, usersPath, uuid, migrationLogger); err != nil {
			migrationLogger.Printf(WARN, "Failed to transfer database users file: %v\n", err)
		}
	}

	migrationLogger.Printf(INFO, "Importing databases on target\n")
	out, err = runEncodedScript(targetClient, fmt.Sprintf(dbImportScript, dumpPath, usersPath))
	if err != nil {
		return fmt.Errorf("failed to import databases on target: %s", strings.TrimSpace(out))
	}
	migrationLogger.Printf(INFO, "Database import result: %s\n", strings.TrimSpace(out))

	// Best-effort cleanup of temp dump files on both hosts.
	_, _ = runSSHCommand(sourceClient, fmt.Sprintf("rm -f '%s' '%s'", dumpPath, usersPath))
	_, _ = runSSHCommand(targetClient, fmt.Sprintf("rm -f '%s' '%s'", dumpPath, usersPath))

	return nil
}
