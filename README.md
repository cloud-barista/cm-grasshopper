# cm-grasshopper

# Software Migration
This repository provides software Level(e.g., Linux Package, Container Image...) migration features. This is a sub-system on Cloud-Barista platform and utilizes CB-Tumblebug to depoly a multi-cloud Software as a target computing Software.

## Overview

Software Migration framework (codename: cm-grasshopper) is going to support:

* Include Linux Pacakge and Container Image, Migrating software from source computing to a target computing environment


<details>
    <summary>Terminology</summary>

* Source Computing  
  The source computing, serving as the target for configuration and information collection, for the migration to multi-cloud
* Target Computing  
  The target computing is migration target as multi-cloud

</details>

## Execution and development environment
* Tested operating systems (OSs):
  * Ubuntu 24.04, Ubuntu 22.04, Ubuntu 18.04, Rocky Linux 9
* Language:
  * Go: 1.24.6

## How to run

### 1. Write the configuration file.
  - Configuration file name is 'cm-grasshopper.yaml'
  - The configuration file must be placed in one of the following directories.
    - .cm-grasshopper/conf directory under user's home directory
      - 'conf' directory where running the binary
    - 'conf' directory where placed in the path of 'CMGRASSHOPPER_ROOT' environment variable
  - Configuration options
    - listen
      - port : Listen port of the API.
    - software
      - temp_folder: Temporary folder while running software migration. (Used for copying Ansible playbook files.)
      - log_folder: Log folder used for logging software installation and migration.
    - ansible
      - playbook_root_path: Root path used for store Ansible playbook files.
    - honeybee
      - server_address : IP address of the honeybee server's API.
      - server_port : Port of the honeybee server's API.
    - tumblebug
        - server_address : IP address of the tumblebug server's API.
        - server_port : Port of the tumblebug server's API.
        - username: Username of the tumblebug authentication
        - password: Password of the tumblebug authentication
  - Configuration file example
    ```yaml
    cm-grasshopper:
        listen:
            port: 8084
    software:
        temp_folder: ./software_temp
        log_folder: ./software_log
    ansible:
        playbook_root_path: ./playbook
    honeybee:
        server_address: 127.0.0.1
        server_port: 8081
    tumblebug:
        server_address: 127.0.0.1
        server_port: 1323
        username: ****
        password: ****
    ```

### 2. Copy honeybee private key file (Not needed if honeybee and grasshopper are running with Docker)
Copy honeybee private key file (honeybee.key) to .cm-grasshopper/ directory under user's home directory or the path of 'CMGRASSHOPPER_ROOT' environment variable.
You can get honeybee.key from .cm-honeybee/ directory under user's home directory or the path of 'CMHONEYBEE_ROOT' environment variable.

If you are running honeybee within Docker, you can copy it with this command.

 ```shell
 docker cp cm-honeybee:/root/.cm-honeybee/honeybee.key .
 ```

### 3. Build and run the binary
 ```shell
 make run
 ```

Or, you can run it within Docker by this command.
 ```shell
 make run_docker
 ```

Docker container will use the default honeybee private key file.
To use the copied honeybee private key file, uncomment it below in the `docker-compose.yaml` file.
```shell
#- ./honeybee.key:/root/.cm-grasshopper/honeybee.key:ro
```

### 4. Import software list from honeybee
```shell
curl -X 'POST' \
  'http://{honeybee IP Address}:8081/honeybee/source_group/{Source Group ID}/import/software' \
  -H 'accept: application/json' \
  -d ''
```

### 5. Get refined software list from honeybee
```shell
curl -X 'GET' \
  'http://{honeybee IP Address}:8081/honeybee/source_group/{Source Group ID}/software/refined' \
  -H 'accept: application/json'
```

<details>
    <summary>Response Example</summary>

```json
{
  "sourceSoftwareModel": {
    "source_group_id": "1b9a25ec-2352-4583-8182-597d18713b29",
    "connection_info_list": [
      {
        "connection_id": "8c90f085-4d6d-427a-ab69-5e1f8e8b04e5",
        "softwares": {
          "binaries": null,
          "packages": [
            {
              "name": "acpid",
              "type": "deb",
              "version": "1:2.0.33-1ubuntu1"
            },
            {
              "name": "alsa-topology-conf",
              "type": "deb",
              "version": "1.2.5.1-2"
            },
            {
              "name": "alsa-ucm-conf",
              "type": "deb",
              "version": "1.2.6.3-1ubuntu1.12"
            },
            {
              "name": "apport-symptoms",
              "type": "deb",
              "version": "0.24"
            },
            {
              "name": "at-spi2-core",
              "type": "deb",
              "version": "2.44.0-3"
            },
            {
              "name": "bash-completion",
              "type": "deb",
              "version": "1:2.11-5ubuntu1"
            },
            {
              "name": "bzip2",
              "type": "deb",
              "version": "1.0.8-5build1"
            },
            {
              "name": "command-not-found",
              "type": "deb",
              "version": "22.04.0"
            },
            {
              "name": "cryptsetup-initramfs",
              "type": "deb",
              "version": "2:2.4.3-1ubuntu1.3"
            },
            {
              "name": "fonts-dejavu-extra",
              "type": "deb",
              "version": "2.37-2build1"
            },
            {
              "name": "friendly-recovery",
              "type": "deb",
              "version": "0.2.42"
            },
            {
              "name": "hsflowd",
              "type": "deb",
              "version": "2.0.53-1 /etc/hsflowd.conf c7504d393334ed92785866d3a5200d9e /etc/dbus-1/system.d/net.sflow.hsflowd.conf 1222fb8baf48e409b4ba870dc0881926"
            },
            {
              "name": "iputils-tracepath",
              "type": "deb",
              "version": "3:20211215-1ubuntu0.1"
            },
            {
              "name": "irqbalance",
              "type": "deb",
              "version": "1.8.0-1ubuntu0.2"
            },
            {
              "name": "landscape-common",
              "type": "deb",
              "version": "23.02-0ubuntu1~22.04.4"
            },
            {
              "name": "libatasmart4",
              "type": "deb",
              "version": "0.19-5build2"
            },
            {
              "name": "libatk-wrapper-java-jni",
              "type": "deb",
              "version": "0.38.0-5build1"
            },
            {
              "name": "libblockdev-crypto2",
              "type": "deb",
              "version": "2.26-1ubuntu0.1"
            },
            {
              "name": "libblockdev-fs2",
              "type": "deb",
              "version": "2.26-1ubuntu0.1"
            },
            {
              "name": "libblockdev-loop2",
              "type": "deb",
              "version": "2.26-1ubuntu0.1"
            },
            {
              "name": "libblockdev-part2",
              "type": "deb",
              "version": "2.26-1ubuntu0.1"
            },
            {
              "name": "libblockdev-swap2",
              "type": "deb",
              "version": "2.26-1ubuntu0.1"
            },
            {
              "name": "libblockdev2",
              "type": "deb",
              "version": "2.26-1ubuntu0.1"
            },
            {
              "name": "libcgi-fast-perl",
              "type": "deb",
              "version": "1:2.15-1"
            },
            {
              "name": "libclone-perl",
              "type": "deb",
              "version": "0.45-1build3"
            },
            {
              "name": "libdbd-mysql-perl",
              "type": "deb",
              "version": "4.050-5ubuntu0.22.04.1"
            },
            {
              "name": "libfcgi-bin",
              "type": "deb",
              "version": "2.4.2-2ubuntu0.1"
            },
            {
              "name": "libflashrom1",
              "type": "deb",
              "version": "1.2-5build1"
            },
            {
              "name": "libfribidi0",
              "type": "deb",
              "version": "1.0.8-2ubuntu3.1"
            },
            {
              "name": "libfwupdplugin5",
              "type": "deb",
              "version": "1.7.9-1~22.04.3"
            },
            {
              "name": "libgl1-amber-dri",
              "type": "deb",
              "version": "21.3.9-0ubuntu1~22.04.1"
            },
            {
              "name": "libhtml-template-perl",
              "type": "deb",
              "version": "2.97-1.1"
            },
            {
              "name": "libhttp-message-perl",
              "type": "deb",
              "version": "6.36-1"
            },
            {
              "name": "libmm-glib0",
              "type": "deb",
              "version": "1.20.0-1~ubuntu22.04.4"
            },
            {
              "name": "libqmi-proxy",
              "type": "deb",
              "version": "1.32.0-1ubuntu0.22.04.1"
            },
            {
              "name": "libsmbios-c2",
              "type": "deb",
              "version": "2.4.3-1build1"
            },
            {
              "name": "libudisks2-0",
              "type": "deb",
              "version": "2.9.4-1ubuntu2.2"
            },
            {
              "name": "libxt-dev",
              "type": "deb",
              "version": "1:1.2.1-1"
            },
            {
              "name": "linux-headers-5.15.0-142-generic",
              "type": "deb",
              "version": "5.15.0-142.152"
            },
            {
              "name": "linux-virtual",
              "type": "deb",
              "version": "5.15.0.152.152"
            },
            {
              "name": "manpages",
              "type": "deb",
              "version": "5.10-1ubuntu1"
            },
            {
              "name": "mariadb-server",
              "type": "deb",
              "version": "1:10.6.22-0ubuntu0.22.04.1"
            },
            {
              "name": "mtr-tiny",
              "type": "deb",
              "version": "0.95-1"
            },
            {
              "name": "nano",
              "type": "deb",
              "version": "6.2-1ubuntu0.1"
            },
            {
              "name": "ncurses-term",
              "type": "deb",
              "version": "6.3-2ubuntu0.1"
            },
            {
              "name": "net-tools",
              "type": "deb",
              "version": "1.60+git20181103.0eebece-1ubuntu5.4"
            },
            {
              "name": "nginx",
              "type": "deb",
              "version": "1.18.0-6ubuntu14.7"
            },
            {
              "name": "ntfs-3g",
              "type": "deb",
              "version": "1:2021.8.22-3ubuntu1.2"
            },
            {
              "name": "open-vm-tools",
              "type": "deb",
              "version": "2:12.3.5-3~ubuntu0.22.04.2"
            },
            {
              "name": "openjdk-11-jdk",
              "type": "deb",
              "version": "11.0.28+6-1ubuntu1~22.04.1"
            },
            {
              "name": "packagekit-tools",
              "type": "deb",
              "version": "1.2.5-2ubuntu3"
            },
            {
              "name": "pastebinit",
              "type": "deb",
              "version": "1.5.1-1ubuntu1"
            },
            {
              "name": "php8.1-curl",
              "type": "deb",
              "version": "8.1.2-1ubuntu2.22"
            },
            {
              "name": "php8.1-fpm",
              "type": "deb",
              "version": "8.1.2-1ubuntu2.22"
            },
            {
              "name": "php8.1-gd",
              "type": "deb",
              "version": "8.1.2-1ubuntu2.22"
            },
            {
              "name": "php8.1-intl",
              "type": "deb",
              "version": "8.1.2-1ubuntu2.22"
            },
            {
              "name": "php8.1-mbstring",
              "type": "deb",
              "version": "8.1.2-1ubuntu2.22"
            },
            {
              "name": "php8.1-mysql",
              "type": "deb",
              "version": "8.1.2-1ubuntu2.22"
            },
            {
              "name": "php8.1-soap",
              "type": "deb",
              "version": "8.1.2-1ubuntu2.22"
            },
            {
              "name": "php8.1-xml",
              "type": "deb",
              "version": "8.1.2-1ubuntu2.22"
            },
            {
              "name": "php8.1-xmlrpc",
              "type": "deb",
              "version": "3:1.0.0~rc3-2"
            },
            {
              "name": "php8.1-zip",
              "type": "deb",
              "version": "8.1.2-1ubuntu2.22"
            },
            {
              "name": "plymouth-theme-ubuntu-text",
              "type": "deb",
              "version": "0.9.5+git20211018-1ubuntu3"
            },
            {
              "name": "powermgmt-base",
              "type": "deb",
              "version": "1.36"
            },
            {
              "name": "publicsuffix",
              "type": "deb",
              "version": "20211207.1025-1"
            },
            {
              "name": "python3-click",
              "type": "deb",
              "version": "8.0.3-1"
            },
            {
              "name": "rsyslog",
              "type": "deb",
              "version": "8.2112.0-2ubuntu2.2"
            },
            {
              "name": "run-one",
              "type": "deb",
              "version": "1.17-0ubuntu1"
            },
            {
              "name": "shared-mime-info",
              "type": "deb",
              "version": "2.1-2"
            },
            {
              "name": "tcpdump",
              "type": "deb",
              "version": "4.99.1-3ubuntu0.2"
            },
            {
              "name": "telnet",
              "type": "deb",
              "version": "0.17-44build1"
            },
            {
              "name": "ubuntu-minimal",
              "type": "deb",
              "version": "1.481.4"
            },
            {
              "name": "ubuntu-server",
              "type": "deb",
              "version": "1.481.4"
            },
            {
              "name": "ubuntu-standard",
              "type": "deb",
              "version": "1.481.4"
            },
            {
              "name": "ufw",
              "type": "deb",
              "version": "0.36.1-4ubuntu0.1"
            },
            {
              "name": "usb.ids",
              "type": "deb",
              "version": "2022.04.02-1"
            },
            {
              "name": "uuid-runtime",
              "type": "deb",
              "version": "2.37.2-4ubuntu3.4"
            },
            {
              "name": "xauth",
              "type": "deb",
              "version": "1:1.1-1build2"
            },
            {
              "name": "xdg-user-dirs",
              "type": "deb",
              "version": "0.17-2ubuntu4"
            }
          ],
          "containers": null,
          "kubernetes": null
        }
      }
    ]
  }
}
```
</details>

### 6. Get software migration list from grasshopper
```shell
curl -X 'POST' \
  'http://127.0.0.1:8084/grasshopper/software/migration_list' \
  -H 'accept: application/json' \
  -H 'Content-Type: application/json' \
  -d '...(Use sourceSoftwareModel in section 5)...'
```

<details>
    <summary>Response Example</summary>

```json
{
  "targetSoftwareModel": {
    "servers": [
      {
        "source_connection_info_id": "8c90f085-4d6d-427a-ab69-5e1f8e8b04e5",
        "migration_list": {
          "binaries": null,
          "packages": [
            {
              "order": 1,
              "name": "acpid",
              "version": "1:2.0.33-1ubuntu1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 2,
              "name": "alsa-topology-conf",
              "version": "1.2.5.1-2",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 3,
              "name": "alsa-ucm-conf",
              "version": "1.2.6.3-1ubuntu1.12",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 4,
              "name": "apport-symptoms",
              "version": "0.24",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 5,
              "name": "at-spi2-core",
              "version": "2.44.0-3",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 6,
              "name": "bash-completion",
              "version": "1:2.11-5ubuntu1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 7,
              "name": "bzip2",
              "version": "1.0.8-5build1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 8,
              "name": "command-not-found",
              "version": "22.04.0",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 9,
              "name": "cryptsetup-initramfs",
              "version": "2:2.4.3-1ubuntu1.3",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 10,
              "name": "fonts-dejavu-extra",
              "version": "2.37-2build1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 11,
              "name": "friendly-recovery",
              "version": "0.2.42",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 12,
              "name": "hsflowd",
              "version": "2.0.53-1 /etc/hsflowd.conf c7504d393334ed92785866d3a5200d9e /etc/dbus-1/system.d/net.sflow.hsflowd.conf 1222fb8baf48e409b4ba870dc0881926",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 13,
              "name": "iputils-tracepath",
              "version": "3:20211215-1ubuntu0.1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 14,
              "name": "irqbalance",
              "version": "1.8.0-1ubuntu0.2",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 15,
              "name": "libatk-wrapper-java-jni",
              "version": "0.38.0-5build1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 16,
              "name": "libcgi-fast-perl",
              "version": "1:2.15-1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 17,
              "name": "libclone-perl",
              "version": "0.45-1build3",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 18,
              "name": "libdbd-mysql-perl",
              "version": "4.050-5ubuntu0.22.04.1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 19,
              "name": "libfcgi-bin",
              "version": "2.4.2-2ubuntu0.1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 20,
              "name": "libgl1-amber-dri",
              "version": "21.3.9-0ubuntu1~22.04.1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 21,
              "name": "libhtml-template-perl",
              "version": "2.97-1.1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 22,
              "name": "libhttp-message-perl",
              "version": "6.36-1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 23,
              "name": "libqmi-proxy",
              "version": "1.32.0-1ubuntu0.22.04.1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 24,
              "name": "linux-virtual",
              "version": "5.15.0.152.152",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 25,
              "name": "manpages",
              "version": "5.10-1ubuntu1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 26,
              "name": "mariadb-server",
              "version": "1:10.6.22-0ubuntu0.22.04.1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 27,
              "name": "mtr-tiny",
              "version": "0.95-1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 28,
              "name": "nano",
              "version": "6.2-1ubuntu0.1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 29,
              "name": "ncurses-term",
              "version": "6.3-2ubuntu0.1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 30,
              "name": "net-tools",
              "version": "1.60+git20181103.0eebece-1ubuntu5.4",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 31,
              "name": "nginx",
              "version": "1.18.0-6ubuntu14.7",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 32,
              "name": "ntfs-3g",
              "version": "1:2021.8.22-3ubuntu1.2",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 33,
              "name": "open-vm-tools",
              "version": "2:12.3.5-3~ubuntu0.22.04.2",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 34,
              "name": "openjdk-11-jdk",
              "version": "11.0.28+6-1ubuntu1~22.04.1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 35,
              "name": "packagekit-tools",
              "version": "1.2.5-2ubuntu3",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 36,
              "name": "pastebinit",
              "version": "1.5.1-1ubuntu1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 37,
              "name": "php8.1-curl",
              "version": "8.1.2-1ubuntu2.22",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 38,
              "name": "php8.1-fpm",
              "version": "8.1.2-1ubuntu2.22",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 39,
              "name": "php8.1-gd",
              "version": "8.1.2-1ubuntu2.22",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 40,
              "name": "php8.1-intl",
              "version": "8.1.2-1ubuntu2.22",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 41,
              "name": "php8.1-mbstring",
              "version": "8.1.2-1ubuntu2.22",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 42,
              "name": "php8.1-mysql",
              "version": "8.1.2-1ubuntu2.22",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 43,
              "name": "php8.1-soap",
              "version": "8.1.2-1ubuntu2.22",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 44,
              "name": "php8.1-xml",
              "version": "8.1.2-1ubuntu2.22",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 45,
              "name": "php8.1-xmlrpc",
              "version": "3:1.0.0~rc3-2",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 46,
              "name": "php8.1-zip",
              "version": "8.1.2-1ubuntu2.22",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 47,
              "name": "plymouth-theme-ubuntu-text",
              "version": "0.9.5+git20211018-1ubuntu3",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 48,
              "name": "powermgmt-base",
              "version": "1.36",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 49,
              "name": "publicsuffix",
              "version": "20211207.1025-1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 50,
              "name": "python3-click",
              "version": "8.0.3-1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 51,
              "name": "rsyslog",
              "version": "8.2112.0-2ubuntu2.2",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 52,
              "name": "run-one",
              "version": "1.17-0ubuntu1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 53,
              "name": "shared-mime-info",
              "version": "2.1-2",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 54,
              "name": "tcpdump",
              "version": "4.99.1-3ubuntu0.2",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 55,
              "name": "telnet",
              "version": "0.17-44build1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 56,
              "name": "ubuntu-minimal",
              "version": "1.481.4",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 57,
              "name": "ubuntu-server",
              "version": "1.481.4",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 58,
              "name": "ubuntu-standard",
              "version": "1.481.4",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 59,
              "name": "ufw",
              "version": "0.36.1-4ubuntu0.1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 60,
              "name": "usb.ids",
              "version": "2022.04.02-1",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 61,
              "name": "uuid-runtime",
              "version": "2.37.2-4ubuntu3.4",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 62,
              "name": "xauth",
              "version": "1:1.1-1build2",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            },
            {
              "order": 63,
              "name": "xdg-user-dirs",
              "version": "0.17-2ubuntu4",
              "needed_packages": [
                ""
              ],
              "need_to_delete_packages": [
                ""
              ],
              "custom_data_paths": [],
              "custom_configs": null,
              "repo_url": "",
              "gpg_key_url": "",
              "repo_use_os_version_code": false
            }
          ],
          "containers": null,
          "kubernetes": null
        },
        "errors": []
      }
    ]
  }
}
```
</details>

### 7. Run software migration from grasshopper
```shell
curl -X 'POST' \
  'http://210.207.104.224:8084/grasshopper/software/migrate?nsId=mig01&mciId=mmci01' \
  -H 'accept: application/json' \
  -H 'Content-Type: application/json' \
  -d '...(Use targetSoftwareModel in section 6)...'
```

- Response
```json
{
  "execution_id": "aa253834-0668-4eda-8416-a0dae9f8c483",
  "target_mappings": [
    {
      "source_connection_info_id": "8c90f085-4d6d-427a-ab69-5e1f8e8b04e5",
      "target": {
        "namespace_id": "mig01",
        "mci_id": "mmci01",
        "vm_id": "migrated-cb95f7e4-570c-468c-af77-2263271bc138-1"
      }
    }
  ]
}
```

### 8. Get software migration log
You can check the logs by providing the execution ID.

```shell
curl -X 'GET' \
  'http://127.0.0.1:8084/grasshopper/software/migrate/log/{execution_id in section 7}' \
  -H 'accept: application/json'
```

<details>
    <summary>Response Example</summary>

```json
{
  "uuid": "aa253834-0668-4eda-8416-a0dae9f8c483",
  "install_log": "\nPLAY [Install nginx] ***********************************************************\n\nTASK [Gathering Facts] *********************************************************\nTuesday 12 November 2024  00:48:28 +0900 (0:00:00.006)       0:00:00.006 ****** \nTuesday 12 November 2024  00:48:28 +0900 (0:00:00.006)       0:00:00.006 ****** \nok: [20.41.114.16]\n\nTASK [role : Remove deb package (Debian family)] *******************************\nTuesday 12 November 2024  00:48:30 +0900 (0:00:02.027)       0:00:02.034 ****** \nTuesday 12 November 2024  00:48:30 +0900 (0:00:02.027)       0:00:02.034 ****** \nskipping: [20.41.114.16]\n\nTASK [role : Remove rpm package (Redhat family)] *******************************\nTuesday 12 November 2024  00:48:30 +0900 (0:00:00.011)       0:00:02.046 ****** \nTuesday 12 November 2024  00:48:30 +0900 (0:00:00.011)       0:00:02.046 ****** \nskipping: [20.41.114.16]\n\nTASK [role : Install deb package (Debian family)] ******************************\nTuesday 12 November 2024  00:48:30 +0900 (0:00:00.010)       0:00:02.056 ****** \nTuesday 12 November 2024  00:48:30 +0900 (0:00:00.010)       0:00:02.056 ****** \nok: [20.41.114.16] => (item=nginx)\n\nTASK [role : Install rpm package (Redhat family)] ******************************\nTuesday 12 November 2024  00:48:35 +0900 (0:00:04.857)       0:00:06.914 ****** \nTuesday 12 November 2024  00:48:35 +0900 (0:00:04.857)       0:00:06.913 ****** \nskipping: [20.41.114.16] => (item=nginx) \nskipping: [20.41.114.16]\n\nPLAY RECAP *********************************************************************\n20.41.114.16               : ok=2    changed=0    unreachable=0    failed=0    skipped=3    rescued=0    ignored=0   \n\nTuesday 12 November 2024  00:48:35 +0900 (0:00:00.034)       0:00:06.948 ****** \n=============================================================================== \nrole : Install deb package (Debian family) ------------------------------ 4.86s\nGathering Facts --------------------------------------------------------- 2.03s\nrole : Install rpm package (Redhat family) ------------------------------ 0.03s\nrole : Remove deb package (Debian family) ------------------------------- 0.01s\nrole : Remove rpm package (Redhat family) ------------------------------- 0.01s\nTuesday 12 November 2024  00:48:35 +0900 (0:00:00.034)       0:00:06.948 ****** \n=============================================================================== \nrole -------------------------------------------------------------------- 4.91s\ngather_facts ------------------------------------------------------------ 2.03s\n~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~ \ntotal ------------------------------------------------------------------- 6.94s\n",
  "migration_log": "2024/11/12 00:48:35 [ INFO ] Starting config copier for package: nginx (UUID: aa253834-0668-4eda-8416-a0dae9f8c483)\n2024/11/12 00:48:35 [ INFO ] Finding configuration files for package: nginx\n2024/11/12 00:48:35 [ INFO ] Starting config search for package: nginx\n2024/11/12 00:48:35 [ DEBUG ] Executing config search command\n2024/11/12 00:48:40 [ DEBUG ] Processing found config files\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/init.d/nginx\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/init.d/nginx, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/init.d/nginx []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/logrotate.d/nginx\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/logrotate.d/nginx, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/logrotate.d/nginx []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/fastcgi.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/fastcgi.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/fastcgi.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/fastcgi_params\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/fastcgi_params, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/fastcgi_params []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/g gg.conf [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/g gg.conf, Status: Custom\n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/g gg.conf [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/ggg.conf [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/ggg.conf, Status: Custom\n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/ggg.conf [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/modules-enabled/50-mod-http-geoip2.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/modules-enabled/50-mod-http-geoip2.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/modules-enabled/50-mod-http-geoip2.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/modules-enabled/50-mod-http-image-filter.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/modules-enabled/50-mod-http-image-filter.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/modules-enabled/50-mod-http-image-filter.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/modules-enabled/50-mod-http-xslt-filter.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/modules-enabled/50-mod-http-xslt-filter.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/modules-enabled/50-mod-http-xslt-filter.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/modules-enabled/50-mod-mail.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/modules-enabled/50-mod-mail.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/modules-enabled/50-mod-mail.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/modules-enabled/50-mod-stream.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/modules-enabled/50-mod-stream.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/modules-enabled/50-mod-stream.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/modules-enabled/70-mod-stream-geoip2.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/modules-enabled/70-mod-stream-geoip2.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/modules-enabled/70-mod-stream-geoip2.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/nginx.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/nginx.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/nginx.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/nginx.conf.bak [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/nginx.conf.bak, Status: Custom\n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/nginx.conf.bak [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/proxy_params\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/proxy_params, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/proxy_params []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/scgi_params\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/scgi_params, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/scgi_params []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/sites-available/default [Modified]\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/sites-available/default, Status: Modified\n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/sites-available/default [Modified]\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/sites-available/default_bak [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/sites-available/default_bak, Status: Custom\n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/sites-available/default_bak [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/sites-available/ssl [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/sites-available/ssl, Status: Custom\n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/sites-available/ssl [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/sites-enabled/default [Modified]\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/sites-enabled/default, Status: Modified\n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/sites-enabled/default [Modified]\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/sites-enabled/ssl [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/sites-enabled/ssl, Status: Custom\n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/sites-enabled/ssl [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/snippets/fastcgi-php.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/snippets/fastcgi-php.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/snippets/fastcgi-php.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/snippets/snakeoil.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/snippets/snakeoil.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/snippets/snakeoil.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/uwsgi_params\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/uwsgi_params, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/uwsgi_params []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/ufw/applications.d/nginx\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/ufw/applications.d/nginx, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/ufw/applications.d/nginx []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /usr/share/doc/nginx/copyright\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /usr/share/doc/nginx/copyright, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /usr/share/doc/nginx/copyright []\n2024/11/12 00:48:40 [ INFO ] Found 26 unique config files\n2024/11/12 00:48:40 [ INFO ] Found 26 configuration files for package nginx\n2024/11/12 00:48:40 [ INFO ] Starting config files copy process for 26 files\n2024/11/12 00:48:40 [ INFO ] Processing config file 1/26: /etc/init.d/nginx\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/init.d/nginx\n2024/11/12 00:48:40 [ INFO ] Processing config file 2/26: /etc/logrotate.d/nginx\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/logrotate.d/nginx\n2024/11/12 00:48:40 [ INFO ] Processing config file 3/26: /etc/nginx/fastcgi.conf\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/fastcgi.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 4/26: /etc/nginx/fastcgi_params\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/fastcgi_params\n2024/11/12 00:48:40 [ INFO ] Processing config file 5/26: /etc/nginx/g gg.conf\n2024/11/12 00:48:40 [ INFO ] Copying config file: /etc/nginx/g gg.conf [Status: Custom]\n2024/11/12 00:48:40 [ INFO ] Starting file copy process for: /etc/nginx/g gg.conf\n2024/11/12 00:48:40 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:40 [ DEBUG ] File stats - Permissions: 644, UID: 1001, GID: 1001, Type: regular empty file\n2024/11/12 00:48:40 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:40 [ INFO ] Copying file to target system\n2024/11/12 00:48:40 [ INFO ] Successfully copied file: /etc/nginx/g gg.conf\n2024/11/12 00:48:40 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/g gg.conf\n2024/11/12 00:48:40 [ INFO ] Finding certificate and key paths in file: /etc/nginx/g gg.conf\n2024/11/12 00:48:40 [ DEBUG ] Executing grep command for cert/key paths\n2024/11/12 00:48:40 [ DEBUG ] Processing grep output for pattern matches\n2024/11/12 00:48:40 [ INFO ] Found 0 potential cert/key paths\n2024/11/12 00:48:40 [ INFO ] Found 0 valid cert/key paths\n2024/11/12 00:48:40 [ DEBUG ] No cert/key files found for: /etc/nginx/g gg.conf\n2024/11/12 00:48:40 [ INFO ] Successfully processed config file: /etc/nginx/g gg.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 6/26: /etc/nginx/ggg.conf\n2024/11/12 00:48:40 [ INFO ] Copying config file: /etc/nginx/ggg.conf [Status: Custom]\n2024/11/12 00:48:40 [ INFO ] Starting file copy process for: /etc/nginx/ggg.conf\n2024/11/12 00:48:40 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:40 [ DEBUG ] File stats - Permissions: 644, UID: 1002, GID: 1002, Type: regular empty file\n2024/11/12 00:48:40 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:40 [ INFO ] Copying file to target system\n2024/11/12 00:48:40 [ INFO ] Successfully copied file: /etc/nginx/ggg.conf\n2024/11/12 00:48:40 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/ggg.conf\n2024/11/12 00:48:40 [ INFO ] Finding certificate and key paths in file: /etc/nginx/ggg.conf\n2024/11/12 00:48:40 [ DEBUG ] Executing grep command for cert/key paths\n2024/11/12 00:48:40 [ DEBUG ] Processing grep output for pattern matches\n2024/11/12 00:48:40 [ INFO ] Found 0 potential cert/key paths\n2024/11/12 00:48:40 [ INFO ] Found 0 valid cert/key paths\n2024/11/12 00:48:40 [ DEBUG ] No cert/key files found for: /etc/nginx/ggg.conf\n2024/11/12 00:48:40 [ INFO ] Successfully processed config file: /etc/nginx/ggg.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 7/26: /etc/nginx/modules-enabled/50-mod-http-geoip2.conf\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/modules-enabled/50-mod-http-geoip2.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 8/26: /etc/nginx/modules-enabled/50-mod-http-image-filter.conf\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/modules-enabled/50-mod-http-image-filter.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 9/26: /etc/nginx/modules-enabled/50-mod-http-xslt-filter.conf\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/modules-enabled/50-mod-http-xslt-filter.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 10/26: /etc/nginx/modules-enabled/50-mod-mail.conf\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/modules-enabled/50-mod-mail.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 11/26: /etc/nginx/modules-enabled/50-mod-stream.conf\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/modules-enabled/50-mod-stream.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 12/26: /etc/nginx/modules-enabled/70-mod-stream-geoip2.conf\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/modules-enabled/70-mod-stream-geoip2.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 13/26: /etc/nginx/nginx.conf\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/nginx.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 14/26: /etc/nginx/nginx.conf.bak\n2024/11/12 00:48:40 [ INFO ] Copying config file: /etc/nginx/nginx.conf.bak [Status: Custom]\n2024/11/12 00:48:40 [ INFO ] Starting file copy process for: /etc/nginx/nginx.conf.bak\n2024/11/12 00:48:40 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:40 [ DEBUG ] File stats - Permissions: 644, UID: 0, GID: 0, Type: regular file\n2024/11/12 00:48:40 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:40 [ INFO ] Copying file to target system\n2024/11/12 00:48:40 [ INFO ] Successfully copied file: /etc/nginx/nginx.conf.bak\n2024/11/12 00:48:40 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/nginx.conf.bak\n2024/11/12 00:48:40 [ INFO ] Finding certificate and key paths in file: /etc/nginx/nginx.conf.bak\n2024/11/12 00:48:40 [ DEBUG ] Executing grep command for cert/key paths\n2024/11/12 00:48:40 [ DEBUG ] Processing grep output for pattern matches\n2024/11/12 00:48:40 [ INFO ] Found 0 potential cert/key paths\n2024/11/12 00:48:40 [ INFO ] Found 0 valid cert/key paths\n2024/11/12 00:48:40 [ DEBUG ] No cert/key files found for: /etc/nginx/nginx.conf.bak\n2024/11/12 00:48:40 [ INFO ] Successfully processed config file: /etc/nginx/nginx.conf.bak\n2024/11/12 00:48:40 [ INFO ] Processing config file 15/26: /etc/nginx/proxy_params\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/proxy_params\n2024/11/12 00:48:40 [ INFO ] Processing config file 16/26: /etc/nginx/scgi_params\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/scgi_params\n2024/11/12 00:48:40 [ INFO ] Processing config file 17/26: /etc/nginx/sites-available/default\n2024/11/12 00:48:40 [ INFO ] Copying config file: /etc/nginx/sites-available/default [Status: Modified]\n2024/11/12 00:48:40 [ INFO ] Starting file copy process for: /etc/nginx/sites-available/default\n2024/11/12 00:48:40 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:40 [ DEBUG ] File stats - Permissions: 644, UID: 0, GID: 0, Type: regular file\n2024/11/12 00:48:40 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:40 [ INFO ] Copying file to target system\n2024/11/12 00:48:41 [ INFO ] Successfully copied file: /etc/nginx/sites-available/default\n2024/11/12 00:48:41 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/sites-available/default\n2024/11/12 00:48:41 [ INFO ] Finding certificate and key paths in file: /etc/nginx/sites-available/default\n2024/11/12 00:48:41 [ DEBUG ] Executing grep command for cert/key paths\n2024/11/12 00:48:41 [ DEBUG ] Processing grep output for pattern matches\n2024/11/12 00:48:41 [ INFO ] Found 0 potential cert/key paths\n2024/11/12 00:48:41 [ INFO ] Found 0 valid cert/key paths\n2024/11/12 00:48:41 [ DEBUG ] No cert/key files found for: /etc/nginx/sites-available/default\n2024/11/12 00:48:41 [ INFO ] Successfully processed config file: /etc/nginx/sites-available/default\n2024/11/12 00:48:41 [ INFO ] Processing config file 18/26: /etc/nginx/sites-available/default_bak\n2024/11/12 00:48:41 [ INFO ] Copying config file: /etc/nginx/sites-available/default_bak [Status: Custom]\n2024/11/12 00:48:41 [ INFO ] Starting file copy process for: /etc/nginx/sites-available/default_bak\n2024/11/12 00:48:41 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:41 [ DEBUG ] File stats - Permissions: 644, UID: 0, GID: 0, Type: regular file\n2024/11/12 00:48:41 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:41 [ INFO ] Copying file to target system\n2024/11/12 00:48:41 [ INFO ] Successfully copied file: /etc/nginx/sites-available/default_bak\n2024/11/12 00:48:41 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/sites-available/default_bak\n2024/11/12 00:48:41 [ INFO ] Finding certificate and key paths in file: /etc/nginx/sites-available/default_bak\n2024/11/12 00:48:41 [ DEBUG ] Executing grep command for cert/key paths\n2024/11/12 00:48:41 [ DEBUG ] Processing grep output for pattern matches\n2024/11/12 00:48:41 [ INFO ] Found 0 potential cert/key paths\n2024/11/12 00:48:41 [ INFO ] Found 0 valid cert/key paths\n2024/11/12 00:48:41 [ DEBUG ] No cert/key files found for: /etc/nginx/sites-available/default_bak\n2024/11/12 00:48:41 [ INFO ] Successfully processed config file: /etc/nginx/sites-available/default_bak\n2024/11/12 00:48:41 [ INFO ] Processing config file 19/26: /etc/nginx/sites-available/ssl\n2024/11/12 00:48:41 [ INFO ] Copying config file: /etc/nginx/sites-available/ssl [Status: Custom]\n2024/11/12 00:48:41 [ INFO ] Starting file copy process for: /etc/nginx/sites-available/ssl\n2024/11/12 00:48:41 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:41 [ DEBUG ] File stats - Permissions: 644, UID: 0, GID: 0, Type: regular file\n2024/11/12 00:48:41 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:41 [ INFO ] Copying file to target system\n2024/11/12 00:48:41 [ INFO ] Successfully copied file: /etc/nginx/sites-available/ssl\n2024/11/12 00:48:41 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/sites-available/ssl\n2024/11/12 00:48:41 [ INFO ] Finding certificate and key paths in file: /etc/nginx/sites-available/ssl\n2024/11/12 00:48:41 [ DEBUG ] Executing grep command for cert/key paths\n2024/11/12 00:48:41 [ DEBUG ] Processing grep output for pattern matches\n2024/11/12 00:48:41 [ DEBUG ] Found unique cert/key path: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:41 [ DEBUG ] Found unique cert/key path: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:41 [ INFO ] Found 2 potential cert/key paths\n2024/11/12 00:48:41 [ DEBUG ] Verifying path existence: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:41 [ INFO ] Verified valid cert/key path: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:41 [ DEBUG ] Verifying path existence: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:41 [ INFO ] Verified valid cert/key path: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:41 [ INFO ] Found 2 valid cert/key paths\n2024/11/12 00:48:41 [ INFO ] Found 2 cert/key files for /etc/nginx/sites-available/ssl\n2024/11/12 00:48:41 [ INFO ] Copying cert/key file 1/2: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:41 [ INFO ] Starting file copy process for: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:41 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:41 [ DEBUG ] File stats - Permissions: 644, UID: 0, GID: 0, Type: regular file\n2024/11/12 00:48:41 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:41 [ INFO ] Copying file to target system\n2024/11/12 00:48:41 [ INFO ] Successfully copied file: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:41 [ INFO ] Successfully copied cert/key file: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:41 [ INFO ] Copying cert/key file 2/2: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:41 [ INFO ] Starting file copy process for: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:41 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:41 [ DEBUG ] File stats - Permissions: 600, UID: 0, GID: 0, Type: regular file\n2024/11/12 00:48:41 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:41 [ INFO ] Copying file to target system\n2024/11/12 00:48:41 [ INFO ] Successfully copied file: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:41 [ INFO ] Successfully copied cert/key file: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:41 [ INFO ] Successfully processed config file: /etc/nginx/sites-available/ssl\n2024/11/12 00:48:41 [ INFO ] Processing config file 20/26: /etc/nginx/sites-enabled/default\n2024/11/12 00:48:41 [ INFO ] Copying config file: /etc/nginx/sites-enabled/default [Status: Modified]\n2024/11/12 00:48:41 [ INFO ] Starting file copy process for: /etc/nginx/sites-enabled/default\n2024/11/12 00:48:41 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:41 [ DEBUG ] File stats - Permissions: 777, UID: 0, GID: 0, Type: symbolic link\n2024/11/12 00:48:41 [ INFO ] File is a symbolic link\n2024/11/12 00:48:41 [ DEBUG ] Symlink target: /etc/nginx/sites-available/default\n2024/11/12 00:48:41 [ INFO ] Copying file to target system\n2024/11/12 00:48:41 [ DEBUG ] Creating symlink on target system\n2024/11/12 00:48:41 [ INFO ] Successfully copied file: /etc/nginx/sites-enabled/default\n2024/11/12 00:48:41 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/sites-enabled/default\n2024/11/12 00:48:41 [ INFO ] Finding certificate and key paths in file: /etc/nginx/sites-enabled/default\n2024/11/12 00:48:41 [ DEBUG ] Executing grep command for cert/key paths\n2024/11/12 00:48:41 [ DEBUG ] Processing grep output for pattern matches\n2024/11/12 00:48:41 [ INFO ] Found 0 potential cert/key paths\n2024/11/12 00:48:41 [ INFO ] Found 0 valid cert/key paths\n2024/11/12 00:48:41 [ DEBUG ] No cert/key files found for: /etc/nginx/sites-enabled/default\n2024/11/12 00:48:41 [ INFO ] Successfully processed config file: /etc/nginx/sites-enabled/default\n2024/11/12 00:48:41 [ INFO ] Processing config file 21/26: /etc/nginx/sites-enabled/ssl\n2024/11/12 00:48:41 [ INFO ] Copying config file: /etc/nginx/sites-enabled/ssl [Status: Custom]\n2024/11/12 00:48:41 [ INFO ] Starting file copy process for: /etc/nginx/sites-enabled/ssl\n2024/11/12 00:48:41 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:41 [ DEBUG ] File stats - Permissions: 777, UID: 0, GID: 0, Type: symbolic link\n2024/11/12 00:48:41 [ INFO ] File is a symbolic link\n2024/11/12 00:48:41 [ DEBUG ] Symlink target: /etc/nginx/sites-available/ssl\n2024/11/12 00:48:42 [ INFO ] Copying file to target system\n2024/11/12 00:48:42 [ DEBUG ] Creating symlink on target system\n2024/11/12 00:48:42 [ INFO ] Successfully copied file: /etc/nginx/sites-enabled/ssl\n2024/11/12 00:48:42 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/sites-enabled/ssl\n2024/11/12 00:48:42 [ INFO ] Finding certificate and key paths in file: /etc/nginx/sites-enabled/ssl\n2024/11/12 00:48:42 [ DEBUG ] Executing grep command for cert/key paths\n2024/11/12 00:48:42 [ DEBUG ] Processing grep output for pattern matches\n2024/11/12 00:48:42 [ DEBUG ] Found unique cert/key path: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:42 [ DEBUG ] Found unique cert/key path: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:42 [ INFO ] Found 2 potential cert/key paths\n2024/11/12 00:48:42 [ DEBUG ] Verifying path existence: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:42 [ INFO ] Verified valid cert/key path: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:42 [ DEBUG ] Verifying path existence: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:42 [ INFO ] Verified valid cert/key path: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:42 [ INFO ] Found 2 valid cert/key paths\n2024/11/12 00:48:42 [ INFO ] Found 2 cert/key files for /etc/nginx/sites-enabled/ssl\n2024/11/12 00:48:42 [ INFO ] Copying cert/key file 1/2: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:42 [ INFO ] Starting file copy process for: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:42 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:42 [ DEBUG ] File stats - Permissions: 644, UID: 0, GID: 0, Type: regular file\n2024/11/12 00:48:42 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:42 [ INFO ] Copying file to target system\n2024/11/12 00:48:42 [ INFO ] Successfully copied file: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:42 [ INFO ] Successfully copied cert/key file: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:42 [ INFO ] Copying cert/key file 2/2: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:42 [ INFO ] Starting file copy process for: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:42 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:42 [ DEBUG ] File stats - Permissions: 600, UID: 0, GID: 0, Type: regular file\n2024/11/12 00:48:42 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:42 [ INFO ] Copying file to target system\n2024/11/12 00:48:42 [ INFO ] Successfully copied file: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:42 [ INFO ] Successfully copied cert/key file: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:42 [ INFO ] Successfully processed config file: /etc/nginx/sites-enabled/ssl\n2024/11/12 00:48:42 [ INFO ] Processing config file 22/26: /etc/nginx/snippets/fastcgi-php.conf\n2024/11/12 00:48:42 [ DEBUG ] Skipping unmodified config: /etc/nginx/snippets/fastcgi-php.conf\n2024/11/12 00:48:42 [ INFO ] Processing config file 23/26: /etc/nginx/snippets/snakeoil.conf\n2024/11/12 00:48:42 [ DEBUG ] Skipping unmodified config: /etc/nginx/snippets/snakeoil.conf\n2024/11/12 00:48:42 [ INFO ] Processing config file 24/26: /etc/nginx/uwsgi_params\n2024/11/12 00:48:42 [ DEBUG ] Skipping unmodified config: /etc/nginx/uwsgi_params\n2024/11/12 00:48:42 [ INFO ] Processing config file 25/26: /etc/ufw/applications.d/nginx\n2024/11/12 00:48:42 [ DEBUG ] Skipping unmodified config: /etc/ufw/applications.d/nginx\n2024/11/12 00:48:42 [ INFO ] Processing config file 26/26: /usr/share/doc/nginx/copyright\n2024/11/12 00:48:42 [ DEBUG ] Skipping unmodified config: /usr/share/doc/nginx/copyright\n2024/11/12 00:48:42 [ INFO ] Completed config files copy process\n2024/11/12 00:48:42 [ INFO ] Successfully completed config copy process for package: nginx\n2024/11/12 00:48:42 [ INFO ] Starting service migration for package: nginx (UUID: aa253834-0668-4eda-8416-a0dae9f8c483)\n2024/11/12 00:48:42 [ DEBUG ] Detecting source system type\n2024/11/12 00:48:42 [ INFO ] Detected system ID: ubuntu\n2024/11/12 00:48:42 [ DEBUG ] Detecting target system type\n2024/11/12 00:48:42 [ INFO ] Detected system ID: ubuntu\n2024/11/12 00:48:42 [ INFO ] Finding services for package: nginx\n2024/11/12 00:48:42 [ INFO ] Found 1 relevant services for package\n2024/11/12 00:48:42 [ INFO ] Getting service info for: nginx\n2024/11/12 00:48:42 [ DEBUG ] Service nginx active status: true\n2024/11/12 00:48:42 [ DEBUG ] Service nginx state: enabled\n2024/11/12 00:48:42 [ DEBUG ] Service nginx enabled status: true\n2024/11/12 00:48:42 [ INFO ] Completed service info gathering for nginx\n2024/11/12 00:48:42 [ INFO ] Completed service discovery with 1 services\n2024/11/12 00:48:42 [ INFO ] Stopping services in dependency order\n2024/11/12 00:48:42 [ INFO ] Stopping service: nginx\n2024/11/12 00:48:42 [ INFO ] Setting service enable/disable states\n2024/11/12 00:48:42 [ INFO ] Enabling service: nginx\n2024/11/12 00:48:43 [ INFO ] Starting services in reverse dependency order\n2024/11/12 00:48:43 [ INFO ] Starting service: nginx\n2024/11/12 00:48:43 [ INFO ] Successfully started service: nginx\n2024/11/12 00:48:43 [ INFO ] Verifying final states\n2024/11/12 00:48:43 [ INFO ] Starting service migration for package: nginx\n2024/11/12 00:48:43 [ DEBUG ] Detecting source PID\n2024/11/12 00:48:43 [ DEBUG ] Source PID: 408710\n2024/11/12 00:48:43 [ DEBUG ] Detecting target PID\n2024/11/12 00:48:43 [ DEBUG ] Target PID: 12630\n2024/11/12 00:48:43 [ DEBUG ] Retrieving source listening connections\n2024/11/12 00:52:41 [ DEBUG ] Source listening connections:\n2024/11/12 00:52:41 [ DEBUG ] - Protocol: tcp, Local Address: 0.0.0.0:8443, Foreign Address: 0.0.0.0:0, PID: 408710, Program: nginx, Command: nginx: master process /usr/sbin/nginx -g daemon on; master_process on; \n2024/11/12 00:52:41 [ DEBUG ] - Protocol: tcp, Local Address: 0.0.0.0:80, Foreign Address: 0.0.0.0:0, PID: 408710, Program: nginx, Command: nginx: master process /usr/sbin/nginx -g daemon on; master_process on; \n2024/11/12 00:52:41 [ DEBUG ] - Protocol: tcp6, Local Address: :::8443, Foreign Address: :::0, PID: 408710, Program: nginx, Command: nginx: master process /usr/sbin/nginx -g daemon on; master_process on; \n2024/11/12 00:52:41 [ DEBUG ] - Protocol: tcp6, Local Address: :::80, Foreign Address: :::0, PID: 408710, Program: nginx, Command: nginx: master process /usr/sbin/nginx -g daemon on; master_process on; \n2024/11/12 00:52:41 [ DEBUG ] Retrieving target listening connections\n2024/11/12 00:52:49 [ DEBUG  Target listening connections:\n2024/11/12 00:52:49 [ DEBUG ] - Protocol: tcp, Local Address: 0.0.0.0:8080, Foreign Address: 0.0.0.0:0, PID: 13514, Program: nginx, Command: nginx: master process /usr/sbin/nginx -g daemon on; master_process on; \n2024/11/12 00:52:49 [ DEBUG ] - Protocol: tcp, Local Address: 0.0.0.0:8443, Foreign Address: 0.0.0.0:0, PID: 13514, Program: nginx, Command: nginx: master process /usr/sbin/nginx -g daemon on; master_process on; \n2024/11/12 00:52:49 [ DEBUG ] - Protocol: tcp6, Local Address: :::8080, Foreign Address: :::0, PID: 13514, Program: nginx, Command: nginx: master process /usr/sbin/nginx -g daemon on; master_process on; \n2024/11/12 00:52:49 [ DEBUG ] - Protocol: tcp6, Local Address: :::8443, Foreign Address: :::0, PID: 13514, Program: nginx, Command: nginx: master process /usr/sbin/nginx -g daemon on; master_process on; \n2024/11/12 00:52:49 [ INFO ] Matching port found: tcp 0.0.0.0:8443\n2024/11/12 00:52:49 [ INFO ] Matching port found: tcp 0.0.0.0:080\n2024/11/12 00:52:49 [ INFO ] Matching port found: tcp6 :::8443\n2024/11/12 00:52:49 [ INFO ] Matching port found: tcp6 :::8080"
}
```
</details>

<details>
    <summary>Install Log Example</summary>

```text

PLAY [Install nginx] ***********************************************************

TASK [Gathering Facts] *********************************************************
Tuesday 12 November 2024  00:48:28 +0900 (0:00:00.006)       0:00:00.006 ****** 
Tuesday 12 November 2024  00:48:28 +0900 (0:00:00.006)       0:00:00.006 ****** 
ok: [20.41.114.16]

TASK [role : Remove deb package (Debian family)] *******************************
Tuesday 12 November 2024  00:48:30 +0900 (0:00:02.027)       0:00:02.034 ****** 
Tuesday 12 November 2024  00:48:30 +0900 (0:00:02.027)       0:00:02.034 ****** 
skipping: [20.41.114.16]

TASK [role : Remove rpm package (Redhat family)] *******************************
Tuesday 12 November 2024  00:48:30 +0900 (0:00:00.011)       0:00:02.046 ****** 
Tuesday 12 November 2024  00:48:30 +0900 (0:00:00.011)       0:00:02.046 ****** 
skipping: [20.41.114.16]

TASK [role : Install deb package (Debian family)] ******************************
Tuesday 12 November 2024  00:48:30 +0900 (0:00:00.010)       0:00:02.056 ****** 
Tuesday 12 November 2024  00:48:30 +0900 (0:00:00.010)       0:00:02.056 ****** 
ok: [20.41.114.16] => (item=nginx)

TASK [role : Install rpm package (Redhat family)] ******************************
Tuesday 12 November 2024  00:48:35 +0900 (0:00:04.857)       0:00:06.914 ****** 
Tuesday 12 November 2024  00:48:35 +0900 (0:00:04.857)       0:00:06.913 ****** 
skipping: [20.41.114.16] => (item=nginx) 
skipping: [20.41.114.16]

PLAY RECAP *********************************************************************
20.41.114.16               : ok=2    changed=0    unreachable=0    failed=0    skipped=3    rescued=0    ignored=0   

Tuesday 12 November 2024  00:48:35 +0900 (0:00:00.034)       0:00:06.948 ****** 
=============================================================================== 
role : Install deb package (Debian family) ------------------------------ 4.86s
Gathering Facts --------------------------------------------------------- 2.03s
role : Install rpm package (Redhat family) ------------------------------ 0.03s
role : Remove deb package (Debian family) ------------------------------- 0.01s
role : Remove rpm package (Redhat family) ------------------------------- 0.01s
Tuesday 12 November 2024  00:48:35 +0900 (0:00:00.034)       0:00:06.948 ****** 
=============================================================================== 
role -------------------------------------------------------------------- 4.91s
gather_facts ------------------------------------------------------------ 2.03s
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~ 
total ------------------------------------------------------------------- 6.94s
```
</details>

<details>
    <summary>Migration Log Example</summary>

```text
2024/11/12 00:49:28 [ INFO ] Starting config copier for package: nginx (UUID: aa253834-0668-4eda-8416-a0dae9f8c483)
2024/11/12 00:49:28 [ INFO ] Finding configuration files for package: nginx
2024/11/12 00:49:28 [ INFO ] Starting config search for package: nginx
2024/11/12 00:49:28 [ DEBUG ] Executing config search command
2024/11/12 00:49:33 [ DEBUG ] Processing found config files
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/init.d/nginx
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/init.d/nginx, Status: 
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/init.d/nginx []
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/logrotate.d/nginx
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/logrotate.d/nginx, Status: 
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/logrotate.d/nginx []
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/fastcgi.conf
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/fastcgi.conf, Status: 
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/fastcgi.conf []
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/fastcgi_params
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/fastcgi_params, Status: 
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/fastcgi_params []
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/g gg.conf [Custom]
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/g gg.conf, Status: Custom
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/g gg.conf [Custom]
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/ggg.conf [Custom]
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/ggg.conf, Status: Custom
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/ggg.conf [Custom]
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/modules-enabled/50-mod-http-geoip2.conf
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/modules-enabled/50-mod-http-geoip2.conf, Status: 
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/modules-enabled/50-mod-http-geoip2.conf []
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/modules-enabled/50-mod-http-image-filter.conf
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/modules-enabled/50-mod-http-image-filter.conf, Status: 
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/modules-enabled/50-mod-http-image-filter.conf []
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/modules-enabled/50-mod-http-xslt-filter.conf
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/modules-enabled/50-mod-http-xslt-filter.conf, Status: 
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/modules-enabled/50-mod-http-xslt-filter.conf []
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/modules-enabled/50-mod-mail.conf
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/modules-enabled/50-mod-mail.conf, Status: 
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/modules-enabled/50-mod-mail.conf []
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/modules-enabled/50-mod-stream.conf
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/modules-enabled/50-mod-stream.conf, Status: 
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/modules-enabled/50-mod-stream.conf []
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/modules-enabled/70-mod-stream-geoip2.conf
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/modules-enabled/70-mod-stream-geoip2.conf, Status: 
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/modules-enabled/70-mod-stream-geoip2.conf []
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/nginx.conf
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/nginx.conf, Status: 
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/nginx.conf []
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/nginx.conf.bak [Custom]
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/nginx.conf.bak, Status: Custom
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/nginx.conf.bak [Custom]
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/proxy_params
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/proxy_params, Status: 
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/proxy_params []
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/scgi_params
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/scgi_params, Status: 
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/scgi_params []
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/sites-available/default [Modified]
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/sites-available/default, Status: Modified
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/sites-available/default [Modified]
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/sites-available/default_bak [Custom]
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/sites-available/default_bak, Status: Custom
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/sites-available/default_bak [Custom]
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/sites-available/ssl [Custom]
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/sites-available/ssl, Status: Custom
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/sites-available/ssl [Custom]
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/sites-enabled/default [Modified]
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/sites-enabled/default, Status: Modified
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/sites-enabled/default [Modified]
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/sites-enabled/ssl [Custom]
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/sites-enabled/ssl, Status: Custom
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/sites-enabled/ssl [Custom]
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/snippets/fastcgi-php.conf
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/snippets/fastcgi-php.conf, Status: 
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/snippets/fastcgi-php.conf []
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/snippets/snakeoil.conf
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/snippets/snakeoil.conf, Status: 
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/snippets/snakeoil.conf []
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/nginx/uwsgi_params
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/nginx/uwsgi_params, Status: 
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/nginx/uwsgi_params []
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /etc/ufw/applications.d/nginx
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /etc/ufw/applications.d/nginx, Status: 
2024/11/12 00:49:33 [ DEBUG ] Found config file: /etc/ufw/applications.d/nginx []
2024/11/12 00:49:33 [ DEBUG ] Parsing config line: /usr/share/doc/nginx/copyright
2024/11/12 00:49:33 [ DEBUG ] Parsed config - Path: /usr/share/doc/nginx/copyright, Status: 
2024/11/12 00:49:33 [ DEBUG ] Found config file: /usr/share/doc/nginx/copyright []
2024/11/12 00:49:33 [ INFO ] Found 26 unique config files
2024/11/12 00:49:33 [ INFO ] Found 26 configuration files for package nginx
2024/11/12 00:49:33 [ INFO ] Starting config files copy process for 26 files
2024/11/12 00:49:33 [ INFO ] Processing config file 1/26: /etc/init.d/nginx
2024/11/12 00:49:33 [ DEBUG ] Skipping unmodified config: /etc/init.d/nginx
2024/11/12 00:49:33 [ INFO ] Processing config file 2/26: /etc/logrotate.d/nginx
2024/11/12 00:49:33 [ DEBUG ] Skipping unmodified config: /etc/logrotate.d/nginx
2024/11/12 00:49:33 [ INFO ] Processing config file 3/26: /etc/nginx/fastcgi.conf
2024/11/12 00:49:33 [ DEBUG ] Skipping unmodified config: /etc/nginx/fastcgi.conf
2024/11/12 00:49:33 [ INFO ] Processing config file 4/26: /etc/nginx/fastcgi_params
2024/11/12 00:49:33 [ DEBUG ] Skipping unmodified config: /etc/nginx/fastcgi_params
2024/11/12 00:49:33 [ INFO ] Processing config file 5/26: /etc/nginx/g gg.conf
2024/11/12 00:49:33 [ INFO ] Copying config file: /etc/nginx/g gg.conf [Status: Custom]
2024/11/12 00:49:33 [ INFO ] Starting file copy process for: /etc/nginx/g gg.conf
2024/11/12 00:49:33 [ DEBUG ] Getting file statistics
2024/11/12 00:49:33 [ DEBUG ] File stats - Permissions: 644, UID: 1001, GID: 1001, Type: regular empty file
2024/11/12 00:49:33 [ DEBUG ] Reading regular file content
2024/11/12 00:49:33 [ INFO ] Copying file to target system
2024/11/12 00:49:33 [ INFO ] Successfully copied file: /etc/nginx/g gg.conf
2024/11/12 00:49:33 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/g gg.conf
2024/11/12 00:49:33 [ INFO ] Finding certificate and key paths in file: /etc/nginx/g gg.conf
2024/11/12 00:49:33 [ DEBUG ] Executing grep command for cert/key paths
2024/11/12 00:49:33 [ DEBUG ] Processing grep output for pattern matches
2024/11/12 00:49:33 [ INFO ] Found 0 potential cert/key paths
2024/11/12 00:49:33 [ INFO ] Found 0 valid cert/key paths
2024/11/12 00:49:33 [ DEBUG ] No cert/key files found for: /etc/nginx/g gg.conf
2024/11/12 00:49:33 [ INFO ] Successfully processed config file: /etc/nginx/g gg.conf
2024/11/12 00:49:33 [ INFO ] Processing config file 6/26: /etc/nginx/ggg.conf
2024/11/12 00:49:33 [ INFO ] Copying config file: /etc/nginx/ggg.conf [Status: Custom]
2024/11/12 00:49:33 [ INFO ] Starting file copy process for: /etc/nginx/ggg.conf
2024/11/12 00:49:33 [ DEBUG ] Getting file statistics
2024/11/12 00:49:33 [ DEBUG ] File stats - Permissions: 644, UID: 1002, GID: 1002, Type: regular empty file
2024/11/12 00:49:33 [ DEBUG ] Reading regular file content
2024/11/12 00:49:33 [ INFO ] Copying file to target system
2024/11/12 00:49:33 [ INFO ] Successfully copied file: /etc/nginx/ggg.conf
2024/11/12 00:49:33 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/ggg.conf
2024/11/12 00:49:33 [ INFO ] Finding certificate and key paths in file: /etc/nginx/ggg.conf
2024/11/12 00:49:33 [ DEBUG ] Executing grep command for cert/key paths
2024/11/12 00:49:34 [ DEBUG ] Processing grep output for pattern matches
2024/11/12 00:49:34 [ INFO ] Found 0 potential cert/key paths
2024/11/12 00:49:34 [ INFO ] Found 0 valid cert/key paths
2024/11/12 00:49:34 [ DEBUG ] No cert/key files found for: /etc/nginx/ggg.conf
2024/11/12 00:49:34 [ INFO ] Successfully processed config file: /etc/nginx/ggg.conf
2024/11/12 00:49:34 [ INFO ] Processing config file 7/26: /etc/nginx/modules-enabled/50-mod-http-geoip2.conf
2024/11/12 00:49:34 [ DEBUG ] Skipping unmodified config: /etc/nginx/modules-enabled/50-mod-http-geoip2.conf
2024/11/12 00:49:34 [ INFO ] Processing config file 8/26: /etc/nginx/modules-enabled/50-mod-http-image-filter.conf
2024/11/12 00:49:34 [ DEBUG ] Skipping unmodified config: /etc/nginx/modules-enabled/50-mod-http-image-filter.conf
2024/11/12 00:49:34 [ INFO ] Processing config file 9/26: /etc/nginx/modules-enabled/50-mod-http-xslt-filter.conf
2024/11/12 00:49:34 [ DEBUG ] Skipping unmodified config: /etc/nginx/modules-enabled/50-mod-http-xslt-filter.conf
2024/11/12 00:49:34 [ INFO ] Processing config file 10/26: /etc/nginx/modules-enabled/50-mod-mail.conf
2024/11/12 00:49:34 [ DEBUG ] Skipping unmodified config: /etc/nginx/modules-enabled/50-mod-mail.conf
2024/11/12 00:49:34 [ INFO ] Processing config file 11/26: /etc/nginx/modules-enabled/50-mod-stream.conf
2024/11/12 00:49:34 [ DEBUG ] Skipping unmodified config: /etc/nginx/modules-enabled/50-mod-stream.conf
2024/11/12 00:49:34 [ INFO ] Processing config file 12/26: /etc/nginx/modules-enabled/70-mod-stream-geoip2.conf
2024/11/12 00:49:34 [ DEBUG ] Skipping unmodified config: /etc/nginx/modules-enabled/70-mod-stream-geoip2.conf
2024/11/12 00:49:34 [ INFO ] Processing config file 13/26: /etc/nginx/nginx.conf
2024/11/12 00:49:34 [ DEBUG ] Skipping unmodified config: /etc/nginx/nginx.conf
2024/11/12 00:49:34 [ INFO ] Processing config file 14/26: /etc/nginx/nginx.conf.bak
2024/11/12 00:49:34 [ INFO ] Copying config file: /etc/nginx/nginx.conf.bak [Status: Custom]
2024/11/12 00:49:34 [ INFO ] Starting file copy process for: /etc/nginx/nginx.conf.bak
2024/11/12 00:49:34 [ DEBUG ] Getting file statistics
2024/11/12 00:49:34 [ DEBUG ] File stats - Permissions: 644, UID: 0, GID: 0, Type: regular file
2024/11/12 00:49:34 [ DEBUG ] Reading regular file content
2024/11/12 00:49:34 [ INFO ] Copying file to target system
2024/11/12 00:49:34 [ INFO ] Successfully copied file: /etc/nginx/nginx.conf.bak
2024/11/12 00:49:34 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/nginx.conf.bak
2024/11/12 00:49:34 [ INFO ] Finding certificate and key paths in file: /etc/nginx/nginx.conf.bak
2024/11/12 00:49:34 [ DEBUG ] Executing grep command for cert/key paths
2024/11/12 00:49:34 [ DEBUG ] Processing grep output for pattern matches
2024/11/12 00:49:34 [ INFO ] Found 0 potential cert/key paths
2024/11/12 00:49:34 [ INFO ] Found 0 valid cert/key paths
2024/11/12 00:49:34 [ DEBUG ] No cert/key files found for: /etc/nginx/nginx.conf.bak
2024/11/12 00:49:34 [ INFO ] Successfully processed config file: /etc/nginx/nginx.conf.bak
2024/11/12 00:49:34 [ INFO ] Processing config file 15/26: /etc/nginx/proxy_params
2024/11/12 00:49:34 [ DEBUG ] Skipping unmodified config: /etc/nginx/proxy_params
2024/11/12 00:49:34 [ INFO ] Processing config file 16/26: /etc/nginx/scgi_params
2024/11/12 00:49:34 [ DEBUG ] Skipping unmodified config: /etc/nginx/scgi_params
2024/11/12 00:49:34 [ INFO ] Processing config file 17/26: /etc/nginx/sites-available/default
2024/11/12 00:49:34 [ INFO ] Copying config file: /etc/nginx/sites-available/default [Status: Modified]
2024/11/12 00:49:34 [ INFO ] Starting file copy process for: /etc/nginx/sites-available/default
2024/11/12 00:49:34 [ DEBUG ] Getting file statistics
2024/11/12 00:49:34 [ DEBUG ] File stats - Permissions: 644, UID: 0, GID: 0, Type: regular file
2024/11/12 00:49:34 [ DEBUG ] Reading regular file content
2024/11/12 00:49:34 [ INFO ] Copying file to target system
2024/11/12 00:49:34 [ INFO ] Successfully copied file: /etc/nginx/sites-available/default
2024/11/12 00:49:34 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/sites-available/default
2024/11/12 00:49:34 [ INFO ] Finding certificate and key paths in file: /etc/nginx/sites-available/default
2024/11/12 00:49:34 [ DEBUG ] Executing grep command for cert/key paths
2024/11/12 00:49:34 [ DEBUG ] Processing grep output for pattern matches
2024/11/12 00:49:34 [ INFO ] Found 0 potential cert/key paths
2024/11/12 00:49:34 [ INFO ] Found 0 valid cert/key paths
2024/11/12 00:49:34 [ DEBUG ] No cert/key files found for: /etc/nginx/sites-available/default
2024/11/12 00:49:34 [ INFO ] Successfully processed config file: /etc/nginx/sites-available/default
2024/11/12 00:49:34 [ INFO ] Processing config file 18/26: /etc/nginx/sites-available/default_bak
2024/11/12 00:49:34 [ INFO ] Copying config file: /etc/nginx/sites-available/default_bak [Status: Custom]
2024/11/12 00:49:34 [ INFO ] Starting file copy process for: /etc/nginx/sites-available/default_bak
2024/11/12 00:49:34 [ DEBUG ] Getting file statistics
2024/11/12 00:49:34 [ DEBUG ] File stats - Permissions: 644, UID: 0, GID: 0, Type: regular file
2024/11/12 00:49:34 [ DEBUG ] Reading regular file content
2024/11/12 00:49:34 [ INFO ] Copying file to target system
2024/11/12 00:49:34 [ INFO ] Successfully copied file: /etc/nginx/sites-available/default_bak
2024/11/12 00:49:34 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/sites-available/default_bak
2024/11/12 00:49:34 [ INFO ] Finding certificate and key paths in file: /etc/nginx/sites-available/default_bak
2024/11/12 00:49:34 [ DEBUG ] Executing grep command for cert/key paths
2024/11/12 00:49:34 [ DEBUG ] Processing grep output for pattern matches
2024/11/12 00:49:34 [ INFO ] Found 0 potential cert/key paths
2024/11/12 00:49:34 [ INFO ] Found 0 valid cert/key paths
2024/11/12 00:49:34 [ DEBUG ] No cert/key files found for: /etc/nginx/sites-available/default_bak
2024/11/12 00:49:34 [ INFO ] Successfully processed config file: /etc/nginx/sites-available/default_bak
2024/11/12 00:49:34 [ INFO ] Processing config file 19/26: /etc/nginx/sites-available/ssl
2024/11/12 00:49:34 [ INFO ] Copying config file: /etc/nginx/sites-available/ssl [Status: Custom]
2024/11/12 00:49:34 [ INFO ] Starting file copy process for: /etc/nginx/sites-available/ssl
2024/11/12 00:49:34 [ DEBUG ] Getting file statistics
2024/11/12 00:49:34 [ DEBUG ] File stats - Permissions: 644, UID: 0, GID: 0, Type: regular file
2024/11/12 00:49:34 [ DEBUG ] Reading regular file content
2024/11/12 00:49:34 [ INFO ] Copying file to target system
2024/11/12 00:49:34 [ INFO ] Successfully copied file: /etc/nginx/sites-available/ssl
2024/11/12 00:49:34 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/sites-available/ssl
2024/11/12 00:49:34 [ INFO ] Finding certificate and key paths in file: /etc/nginx/sites-available/ssl
2024/11/12 00:49:34 [ DEBUG ] Executing grep command for cert/key paths
2024/11/12 00:49:34 [ DEBUG ] Processing grep output for pattern matches
2024/11/12 00:49:34 [ DEBUG ] Found unique cert/key path: /etc/nginx/certs/innogrid.com.crt
2024/11/12 00:49:34 [ DEBUG ] Found unique cert/key path: /etc/nginx/certs/innogrid.com.key
2024/11/12 00:49:34 [ INFO ] Found 2 potential cert/key paths
2024/11/12 00:49:34 [ DEBUG ] Verifying path existence: /etc/nginx/certs/innogrid.com.crt
2024/11/12 00:49:34 [ INFO ] Verified valid cert/key path: /etc/nginx/certs/innogrid.com.crt
2024/11/12 00:49:34 [ DEBUG ] Verifying path existence: /etc/nginx/certs/innogrid.com.key
2024/11/12 00:49:34 [ INFO ] Verified valid cert/key path: /etc/nginx/certs/innogrid.com.key
2024/11/12 00:49:34 [ INFO ] Found 2 valid cert/key paths
2024/11/12 00:49:34 [ INFO ] Found 2 cert/key files for /etc/nginx/sites-available/ssl
2024/11/12 00:49:34 [ INFO ] Copying cert/key file 1/2: /etc/nginx/certs/innogrid.com.crt
2024/11/12 00:49:34 [ INFO ] Starting file copy process for: /etc/nginx/certs/innogrid.com.crt
2024/11/12 00:49:34 [ DEBUG ] Getting file statistics
2024/11/12 00:49:34 [ DEBUG ] File stats - Permissions: 644, UID: 0, GID: 0, Type: regular file
2024/11/12 00:49:34 [ DEBUG ] Reading regular file content
2024/11/12 00:49:34 [ INFO ] Copying file to target system
2024/11/12 00:49:34 [ INFO ] Successfully copied file: /etc/nginx/certs/innogrid.com.crt
2024/11/12 00:49:34 [ INFO ] Successfully copied cert/key file: /etc/nginx/certs/innogrid.com.crt
2024/11/12 00:49:34 [ INFO ] Copying cert/key file 2/2: /etc/nginx/certs/innogrid.com.key
2024/11/12 00:49:34 [ INFO ] Starting file copy process for: /etc/nginx/certs/innogrid.com.key
2024/11/12 00:49:34 [ DEBUG ] Getting file statistics
2024/11/12 00:49:34 [ DEBUG ] File stats - Permissions: 600, UID: 0, GID: 0, Type: regular file
2024/11/12 00:49:34 [ DEBUG ] Reading regular file content
2024/11/12 00:49:34 [ INFO ] Copying file to target system
2024/11/12 00:49:35 [ INFO ] Successfully copied file: /etc/nginx/certs/innogrid.com.key
2024/11/12 00:49:35 [ INFO ] Successfully copied cert/key file: /etc/nginx/certs/innogrid.com.key
2024/11/12 00:49:35 [ INFO ] Successfully processed config file: /etc/nginx/sites-available/ssl
2024/11/12 00:49:35 [ INFO ] Processing config file 20/26: /etc/nginx/sites-enabled/default
2024/11/12 00:49:35 [ INFO ] Copying config file: /etc/nginx/sites-enabled/default [Status: Modified]
2024/11/12 00:49:35 [ INFO ] Starting file copy process for: /etc/nginx/sites-enabled/default
2024/11/12 00:49:35 [ DEBUG ] Getting file statistics
2024/11/12 00:49:35 [ DEBUG ] File stats - Permissions: 777, UID: 0, GID: 0, Type: symbolic link
2024/11/12 00:49:35 [ INFO ] File is a symbolic link
2024/11/12 00:49:35 [ DEBUG ] Symlink target: /etc/nginx/sites-available/default
2024/11/12 00:49:35 [ INFO ] Copying file to target system
2024/11/12 00:49:35 [ DEBUG ] Creating symlink on target system
2024/11/12 00:49:35 [ INFO ] Successfully copied file: /etc/nginx/sites-enabled/default
2024/11/12 00:49:35 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/sites-enabled/default
2024/11/12 00:49:35 [ INFO ] Finding certificate and key paths in file: /etc/nginx/sites-enabled/default
2024/11/12 00:49:35 [ DEBUG ] Executing grep command for cert/key paths
2024/11/12 00:49:35 [ DEBUG ] Processing grep output for pattern matches
2024/11/12 00:49:35 [ INFO ] Found 0 potential cert/key paths
2024/11/12 00:49:35 [ INFO ] Found 0 valid cert/key paths
2024/11/12 00:49:35 [ DEBUG ] No cert/key files found for: /etc/nginx/sites-enabled/default
2024/11/12 00:49:35 [ INFO ] Successfully processed config file: /etc/nginx/sites-enabled/default
2024/11/12 00:49:35 [ INFO ] Processing config file 21/26: /etc/nginx/sites-enabled/ssl
2024/11/12 00:49:35 [ INFO ] Copying config file: /etc/nginx/sites-enabled/ssl [Status: Custom]
2024/11/12 00:49:35 [ INFO ] Starting file copy process for: /etc/nginx/sites-enabled/ssl
2024/11/12 00:49:35 [ DEBUG ] Getting file statistics
2024/11/12 00:49:35 [ DEBUG ] File stats - Permissions: 777, UID: 0, GID: 0, Type: symbolic link
2024/11/12 00:49:35 [ INFO ] File is a symbolic link
2024/11/12 00:49:35 [ DEBUG ] Symlink target: /etc/nginx/sites-available/ssl
2024/11/12 00:49:35 [ INFO ] Copying file to target system
2024/11/12 00:49:35 [ DEBUG ] Creating symlink on target system
2024/11/12 00:49:35 [ INFO ] Successfully copied file: /etc/nginx/sites-enabled/ssl
2024/11/12 00:49:35 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/sites-enabled/ssl
2024/11/12 00:49:35 [ INFO ] Finding certificate and key paths in file: /etc/nginx/sites-enabled/ssl
2024/11/12 00:49:35 [ DEBUG ] Executing grep command for cert/key paths
2024/11/12 00:49:35 [ DEBUG ] Processing grep output for pattern matches
2024/11/12 00:49:35 [ DEBUG ] Found unique cert/key path: /etc/nginx/certs/innogrid.com.crt
2024/11/12 00:49:35 [ DEBUG ] Found unique cert/key path: /etc/nginx/certs/innogrid.com.key
2024/11/12 00:49:35 [ INFO ] Found 2 potential cert/key paths
2024/11/12 00:49:35 [ DEBUG ] Verifying path existence: /etc/nginx/certs/innogrid.com.crt
2024/11/12 00:49:35 [ INFO ] Verified valid cert/key path: /etc/nginx/certs/innogrid.com.crt
2024/11/12 00:49:35 [ DEBUG ] Verifying path existence: /etc/nginx/certs/innogrid.com.key
2024/11/12 00:49:35 [ INFO ] Verified valid cert/key path: /etc/nginx/certs/innogrid.com.key
2024/11/12 00:49:35 [ INFO ] Found 2 valid cert/key paths
2024/11/12 00:49:35 [ INFO ] Found 2 cert/key files for /etc/nginx/sites-enabled/ssl
2024/11/12 00:49:35 [ INFO ] Copying cert/key file 1/2: /etc/nginx/certs/innogrid.com.crt
2024/11/12 00:49:35 [ INFO ] Starting file copy process for: /etc/nginx/certs/innogrid.com.crt
2024/11/12 00:49:35 [ DEBUG ] Getting file statistics
2024/11/12 00:49:35 [ DEBUG ] File stats - Permissions: 644, UID: 0, GID: 0, Type: regular file
2024/11/12 00:49:35 [ DEBUG ] Reading regular file content
2024/11/12 00:49:35 [ INFO ] Copying file to target system
2024/11/12 00:49:35 [ INFO ] Successfully copied file: /etc/nginx/certs/innogrid.com.crt
2024/11/12 00:49:35 [ INFO ] Successfully copied cert/key file: /etc/nginx/certs/innogrid.com.crt
2024/11/12 00:49:35 [ INFO ] Copying cert/key file 2/2: /etc/nginx/certs/innogrid.com.key
2024/11/12 00:49:35 [ INFO ] Starting file copy process for: /etc/nginx/certs/innogrid.com.key
2024/11/12 00:49:35 [ DEBUG ] Getting file statistics
2024/11/12 00:49:35 [ DEBUG ] File stats - Permissions: 600, UID: 0, GID: 0, Type: regular file
2024/11/12 00:49:35 [ DEBUG ] Reading regular file content
2024/11/12 00:49:35 [ INFO ] Copying file to target system
2024/11/12 00:49:35 [ INFO ] Successfully copied file: /etc/nginx/certs/innogrid.com.key
2024/11/12 00:49:35 [ INFO ] Successfully copied cert/key file: /etc/nginx/certs/innogrid.com.key
2024/11/12 00:49:35 [ INFO ] Successfully processed config file: /etc/nginx/sites-enabled/ssl
2024/11/12 00:49:35 [ INFO ] Processing config file 22/26: /etc/nginx/snippets/fastcgi-php.conf
2024/11/12 00:49:35 [ DEBUG ] Skipping unmodified config: /etc/nginx/snippets/fastcgi-php.conf
2024/11/12 00:49:35 [ INFO ] Processing config file 23/26: /etc/nginx/snippets/snakeoil.conf
2024/11/12 00:49:35 [ DEBUG ] Skipping unmodified config: /etc/nginx/snippets/snakeoil.conf
2024/11/12 00:49:35 [ INFO ] Processing config file 24/26: /etc/nginx/uwsgi_params
2024/11/12 00:49:35 [ DEBUG ] Skipping unmodified config: /etc/nginx/uwsgi_params
2024/11/12 00:49:35 [ INFO ] Processing config file 25/26: /etc/ufw/applications.d/nginx
2024/11/12 00:49:35 [ DEBUG ] Skipping unmodified config: /etc/ufw/applications.d/nginx
2024/11/12 00:49:35 [ INFO ] Processing config file 26/26: /usr/share/doc/nginx/copyright
2024/11/12 00:49:35 [ DEBUG ] Skipping unmodified config: /usr/share/doc/nginx/copyright
2024/11/12 00:49:35 [ INFO ] Completed config files copy process
2024/11/12 00:49:35 [ INFO ] Successfully completed config copy process for package: nginx
2024/11/12 00:49:35 [ INFO ] Starting service migration for package: nginx (UUID: db646495-c5ee-4bde-a7fc-634f5594992c)
2024/11/12 00:49:35 [ DEBUG ] Detecting source system type
2024/11/12 00:49:35 [ INFO ] Detected system ID: ubuntu
2024/11/12 00:49:35 [ DEBUG ] Detecting target system type
2024/11/12 00:49:35 [ INFO ] Detected system ID: ubuntu
2024/11/12 00:49:35 [ INFO ] Finding services for package: nginx
2024/11/12 00:49:35 [ INFO ] Found 1 relevant services for package
2024/11/12 00:49:35 [ INFO ] Getting service info for: nginx
2024/11/12 00:49:35 [ DEBUG ] Service nginx active status: true
2024/11/12 00:49:35 [ DEBUG ] Service nginx state: enabled
2024/11/12 00:49:35 [ DEBUG ] Service nginx enabled status: true
2024/11/12 00:49:35 [ INFO ] Completed service info gathering for nginx
2024/11/12 00:49:35 [ INFO ] Completed service discovery with 1 services
2024/11/12 00:49:35 [ INFO ] Stopping services in dependency order
2024/11/12 00:49:35 [ INFO ] Stopping service: nginx
2024/11/12 00:49:35 [ INFO ] Setting service enable/disable states
2024/11/12 00:49:36 [ INFO ] Enabling service: nginx
2024/11/12 00:49:36 [ INFO ] Starting services in reverse dependency order
2024/11/12 00:49:36 [ INFO ] Starting service: nginx
2024/11/12 00:49:36 [ INFO ] Successfully started service: nginx
2024/11/12 00:49:36 [ INFO ] Verifying final states
2024/11/12 00:49:36 [ INFO ] Starting service migration for package: nginx
2024/11/12 00:49:36 [ DEBUG ] Detecting source PID
2024/11/12 00:49:36 [ DEBUG ] Source PID: 408710
2024/11/12 00:49:36 [ DEBUG ] Detecting target PID
2024/11/12 00:49:36 [ DEBUG ] Target PID: 13514
2024/11/12 00:49:36 [ DEBUG ] Retrieving source listening connections
2024/11/12 00:52:41 [ DEBUG ] Source listening connections:
2024/11/12 00:52:41 [ DEBUG ] - Protocol: tcp, Local Address: 0.0.0.0:8443, Foreign Address: 0.0.0.0:0, PID: 408710, Program: nginx, Command: nginx: master process /usr/sbin/nginx -g daemon on; master_process on; 
2024/11/12 00:52:41 [ DEBUG ] - Protocol: tcp, Local Address: 0.0.0.0:80, Foreign Address: 0.0.0.0:0, PID: 408710, Program: nginx, Command: nginx: master process /usr/sbin/nginx -g daemon on; master_process on; 
2024/11/12 00:52:41 [ DEBUG ] - Protocol: tcp6, Local Address: :::8443, Foreign Address: :::0, PID: 408710, Program: nginx, Command: nginx: master process /usr/sbin/nginx -g daemon on; master_process on; 
2024/11/12 00:52:41 [ DEBUG ] - Protocol: tcp6, Local Address: :::80, Foreign Address: :::0, PID: 408710, Program: nginx, Command: nginx: master process /usr/sbin/nginx -g daemon on; master_process on; 
2024/11/12 00:52:41 [ DEBUG ] Retrieving target listening connections
2024/11/12 00:52:49 [ DEBUG ] Target listening connections:
2024/11/12 00:52:49 [ DEBUG ] - Protocol: tcp, Local Address: 0.0.0.0:8080, Foreign Address: 0.0.0.0:0, PID: 13514, Program: nginx, Command: nginx: master process /usr/sbin/nginx -g daemon on; master_process on; 
2024/11/12 00:52:49 [ DEBUG ] - Protocol: tcp, Local Address: 0.0.0.0:8443, Foreign Address: 0.0.0.0:0, PID: 13514, Program: nginx, Command: nginx: master process /usr/sbin/nginx -g daemon on; master_process on; 
2024/11/12 00:52:49 [ DEBUG ] - Protocol: tcp6, Local Address: :::8080, Foreign Address: :::0, PID: 13514, Program: nginx, Command: nginx: master process /usr/sbin/nginx -g daemon on; master_process on; 
2024/11/12 00:52:49 [ DEBUG ] - Protocol: tcp6, Local Address: :::8443, Foreign Address: :::0, PID: 13514, Program: nginx, Command: nginx: master process /usr/sbin/nginx -g daemon on; master_process on; 
2024/11/12 00:52:49 [ INFO ] Matching port found: tcp 0.0.0.0:8443
2024/11/12 00:52:49 [ INFO ] Matching port found: tcp 0.0.0.0:8080
2024/11/12 00:52:49 [ INFO ] Matching port found: tcp6 :::8443
2024/11/12 00:52:49 [ INFO ] Matching port found: tcp6 :::8080
```
</details>

## Health-check

Check if CM-Grasshopper is running

```bash
curl http://localhost:8084/grasshopper/readyz

# Output if it's running successfully
# {"message":"CM-Grasshopper API server is ready"}
```


## Check out all APIs
* [Grasshopper APIs (Swagger Document)](https://cloud-barista.github.io/cb-tumblebug-api-web/?url=https://raw.githubusercontent.com/cloud-barista/cm-grasshopper/main/pkg/api/rest/docs/swagger.yaml)
