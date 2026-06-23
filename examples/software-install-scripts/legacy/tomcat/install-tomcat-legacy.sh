#!/bin/bash

## Apache Tomcat Legacy (manual) Install Script by ish
## Working with Ubuntu 22.04 and Ubuntu 24.04
##
## Downloads JDK 21 (Eclipse Temurin) and the latest Apache Tomcat as tarballs,
## installs both manually under /opt, registers a systemd service, and serves a
## status page. This mimics a "legacy" software install (not from the package
## manager) so it can be used as a Source for binary migration testing.

set -e

if [ "$EUID" -ne 0 ]; then
  echo "[!] Please run as root or use sudo!"
  exit 1
fi

OPT_DIR="/opt"
JDK_LINK="${OPT_DIR}/jdk"
TOMCAT_LINK="${OPT_DIR}/tomcat"
APACHE_MIRROR="https://dlcdn.apache.org"
TOMCAT_MAJOR=11   # latest GA major (requires Java 17+, JDK 21 used here)

apt-get update -y
apt-get install -y wget curl tar

# Detect CPU architecture for the JDK download (Temurin naming)
case "$(uname -m)" in
  x86_64)        TEMURIN_ARCH="x64" ;;
  aarch64|arm64) TEMURIN_ARCH="aarch64" ;;
  *) echo "[!] Unsupported architecture: $(uname -m)"; exit 1 ;;
esac

############################################
# 1) Install JDK 21 manually (tarball)
############################################
echo "[*] Downloading JDK 21 (Eclipse Temurin, ${TEMURIN_ARCH})..."
cd /tmp
rm -f jdk21.tar.gz
curl -fL -o jdk21.tar.gz \
  "https://api.adoptium.net/v3/binary/latest/21/ga/linux/${TEMURIN_ARCH}/jdk/hotspot/normal/eclipse"

tar xf jdk21.tar.gz -C "$OPT_DIR"
JDK_DIR=$(find "$OPT_DIR" -maxdepth 1 -type d -name 'jdk-21*' | sort -V | tail -1)
if [ -z "$JDK_DIR" ]; then
  echo "[!] Failed to extract JDK 21."
  exit 1
fi
ln -sfn "$JDK_DIR" "$JDK_LINK"

# Make JAVA_HOME available system-wide
cat << EOF > /etc/profile.d/jdk.sh
export JAVA_HOME=${JDK_LINK}
export PATH=\$JAVA_HOME/bin:\$PATH
EOF
echo "[*] JDK installed: $("${JDK_LINK}/bin/java" -version 2>&1 | head -1)"

############################################
# 2) Install Tomcat manually (tarball)
############################################
echo "[*] Detecting latest Tomcat ${TOMCAT_MAJOR} version..."
TOMCAT_VERSION=$(curl -fsSL "${APACHE_MIRROR}/tomcat/tomcat-${TOMCAT_MAJOR}/" \
  | grep -oE "v${TOMCAT_MAJOR}\.[0-9]+\.[0-9]+/" \
  | sed -E 's#v(.*)/#\1#' | sort -V | uniq | tail -1)

if [ -z "$TOMCAT_VERSION" ]; then
  echo "[!] Failed to detect Tomcat version from the mirror."
  exit 1
fi
echo "[*] tomcat=${TOMCAT_VERSION}"

cd /tmp
rm -f "apache-tomcat-${TOMCAT_VERSION}.tar.gz"
wget -q "${APACHE_MIRROR}/tomcat/tomcat-${TOMCAT_MAJOR}/v${TOMCAT_VERSION}/bin/apache-tomcat-${TOMCAT_VERSION}.tar.gz"

# Remove a previous install of the same version if present, then extract
rm -rf "${OPT_DIR}/apache-tomcat-${TOMCAT_VERSION}"
tar xf "apache-tomcat-${TOMCAT_VERSION}.tar.gz" -C "$OPT_DIR"
ln -sfn "${OPT_DIR}/apache-tomcat-${TOMCAT_VERSION}" "$TOMCAT_LINK"

# Create a dedicated service account
if ! id tomcat &>/dev/null; then
  useradd -r -m -U -d "$TOMCAT_LINK" -s /usr/sbin/nologin tomcat
fi

# Write the status page that proves Tomcat is serving (replace default ROOT app)
rm -rf "${TOMCAT_LINK}/webapps/ROOT"
mkdir -p "${TOMCAT_LINK}/webapps/ROOT"
cat << 'EOF' > "${TOMCAT_LINK}/webapps/ROOT/index.jsp"
<%@ page contentType="text/html; charset=UTF-8" pageEncoding="UTF-8" %>
<!DOCTYPE html>
<html lang="ko">
<head>
    <title>Tomcat 테스트</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 50px auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 { color: #333; }
        .info {
            background: #e8f4f8;
            padding: 15px;
            border-radius: 4px;
            margin: 20px 0;
        }
        code { background: #eee; padding: 2px 6px; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 Tomcat이 정상 작동 중입니다!</h1>
        <div class="info">
            <p><strong>설치 방식:</strong> tar.gz 수동 설치 (패키지 매니저 미사용)</p>
            <p><strong>현재 시간:</strong> <%= new java.util.Date() %></p>
            <p><strong>서버 정보:</strong> <%= application.getServerInfo() %></p>
            <p><strong>JVM:</strong> <%= System.getProperty("java.version") %></p>
            <p><strong>세션 ID:</strong> <%= session.getId() %></p>
        </div>
        <p>이 페이지가 보이면 Legacy 마이그레이션 테스트용 Source의 Tomcat이 성공적으로 구동된 것입니다.</p>
    </div>
</body>
</html>
EOF

# Ownership and executable permissions
chown -R tomcat:tomcat "${OPT_DIR}/apache-tomcat-${TOMCAT_VERSION}"
chmod +x "${TOMCAT_LINK}/bin/"*.sh

# Register a systemd service so the start mechanism is reproducible on migration
cat << EOF > /etc/systemd/system/tomcat.service
[Unit]
Description=Apache Tomcat ${TOMCAT_VERSION} (manual install)
After=network.target

[Service]
Type=forking
User=tomcat
Group=tomcat
Environment=JAVA_HOME=${JDK_LINK}
Environment=CATALINA_HOME=${TOMCAT_LINK}
Environment=CATALINA_BASE=${TOMCAT_LINK}
Environment=CATALINA_PID=${TOMCAT_LINK}/temp/tomcat.pid
Environment=CATALINA_OPTS=-Xms512m -Xmx1024m
ExecStart=${TOMCAT_LINK}/bin/startup.sh
ExecStop=${TOMCAT_LINK}/bin/shutdown.sh
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable tomcat
systemctl restart tomcat

# Verify
echo "[*] Waiting for Tomcat to come up..."
for i in $(seq 1 30); do
  if curl -fsS http://localhost:8080/ 2>/dev/null | grep -q "Tomcat"; then
    echo "[+] Tomcat is serving on http://localhost:8080/ (port 8080)"
    exit 0
  fi
  sleep 1
done

echo "[!] Tomcat did not respond as expected. Check: systemctl status tomcat"
exit 1
