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
  * Go: 1.23.0

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
    - honeybee
      - server_address : IP address of the honeybee server's API.
      - server_port : Port of the honeybee server's API.
  - Configuration file example
    ```yaml
    cm-grasshopper:
        listen:
            port: 8084
        honeybee:
            server_address: 127.0.0.1
            server_port: 8081
    ```

### 2. Copy honeybee private key file
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

### 4. Get software list
```shell
curl -X 'GET' \
  'http://127.0.0.1:8084/grasshopper/software' \
  -H 'accept: application/json'
```

<details>
    <summary>Response Example</summary>

```json
[
  {
    "uuid": "78d3664e-3eb7-4d37-bf8b-b57b7a238693",
    "install_type": "package",
    "name": "docker",
    "version": "latest",
    "os": "Ubuntu",
    "os_version": "22.04",
    "architecture": "x86_64",
    "match_names": "docker,docker-ce,docker.io",
    "needed_packages": "docker-ce,docker-ce-cli,containerd.io,docker-buildx-plugin,docker-compose-plugin",
    "need_to_delete_packages": "docker.io,docker-doc,docker-compose,docker-compose-v2,podman-docker,containerd,runc",
    "repo_url": "https://download.docker.com/linux/ubuntu",
    "gpg_key_url": "https://download.docker.com/linux/ubuntu/gpg",
    "repo_use_os_version_code": true,
    "created_at": "2024-11-04T19:04:03.192747727+09:00",
    "updated_at": "2024-11-04T19:04:03.192747727+09:00"
  },
  {
    "uuid": "aaf49384-1a7c-4b91-9fdc-c7c46aed0882",
    "install_type": "package",
    "name": "nfs-kernel-server",
    "version": "latest",
    "os": "Ubuntu",
    "os_version": "22.04",
    "architecture": "x86_64",
    "match_names": "nfs-server,nfs-kernel-server",
    "needed_packages": "nfs-kernel-server",
    "need_to_delete_packages": "",
    "repo_url": "",
    "gpg_key_url": "",
    "repo_use_os_version_code": false,
    "created_at": "2024-11-04T20:15:00.877746444+09:00",
    "updated_at": "2024-11-04T20:15:00.877746444+09:00"
  },
  {
    "uuid": "aa34795f-3401-4c28-bbe9-157a5788fd75",
    "install_type": "package",
    "name": "nginx",
    "version": "latest",
    "os": "Ubuntu",
    "os_version": "22.04",
    "architecture": "x86_64",
    "match_names": "nginx",
    "needed_packages": "nginx",
    "need_to_delete_packages": "",
    "repo_url": "",
    "gpg_key_url": "",
    "repo_use_os_version_code": false,
    "created_at": "2024-11-04T20:19:03.437025609+09:00",
    "updated_at": "2024-11-04T20:19:03.437025609+09:00"
  }
]
```
</details>

### 5. Register the software (Optional)
```shell
curl -X 'POST' \
  'http://127.0.0.1:8084/grasshopper/software/register' \
  -H 'accept: application/json' \
  -H 'Content-Type: application/json' \
  -d '{ENTER_REQUEST_BODY_HERE}'
```

- Below is a request body example of registering a nginx package.
```json
{
  "architecture": "x86_64",
  "install_type": "package",
  "match_names": [
    "nginx"
  ],
  "name": "nginx",
  "needed_packages": [
    "nginx"
  ],
  "os": "Ubuntu",
  "os_version": "22.04",
  "version": "latest"
}
```

- Below is a request body example of registering a Docker package.
```json
{
  "architecture": "x86_64",
  "gpg_key_url": "https://download.docker.com/linux/ubuntu/gpg",
  "install_type": "package",
  "match_names": [
    "docker",
    "docker-ce",
    "docker.io"
  ],
  "name": "docker",
  "need_to_delete_packages": [
    "docker.io",
    "docker-doc",
    "docker-compose",
    "docker-compose-v2",
    "podman-docker",
    "containerd",
    "runc"
  ],
  "needed_packages": [
    "docker-ce",
    "docker-ce-cli",
    "containerd.io",
    "docker-buildx-plugin",
    "docker-compose-plugin"
  ],
  "os": "Ubuntu",
  "os_version": "22.04",
  "repo_url": "https://download.docker.com/linux/ubuntu",
  "repo_use_os_version_code": true,
  "version": "latest"
}
```

### 6. Get software migration list
You can get a list of software migration list supported by Grasshopper through the source group ID generated by Honeybee.

This list provides a compilation of packages that can be migrated through Grasshopper from those installed in the source computing environment.

```shell
curl -X 'GET' \
  'http://127.0.0.1:8084/grasshopper/software/migration_list/{Source Group ID}' \
  -H 'accept: application/json'
```

<details>
    <summary>Response Example</summary>

```json
{
  "server": [
    {
      "connection_info_id": "829e9c15-a24c-4c39-9e1b-162fcae8f21b",
      "migration_list": [
        {
          "order": 1,
          "software_id": "78d3664e-3eb7-4d37-bf8b-b57b7a238693",
          "software_name": "docker",
          "software_version": "latest",
          "software_install_type": "package"
        },
        {
          "order": 2,
          "software_id": "aaf49384-1a7c-4b91-9fdc-c7c46aed0882",
          "software_name": "nfs-kernel-server",
          "software_version": "latest",
          "software_install_type": "package"
        }
      ],
      "errors": []
    },
    {
      "connection_info_id": "d0b6a2a6-4cd8-4b36-ba41-a5f9a7aeef26",
      "migration_list": [
        {
          "order": 1,
          "software_id": "78d3664e-3eb7-4d37-bf8b-b57b7a238693",
          "software_name": "docker",
          "software_version": "latest",
          "software_install_type": "package"
        },
        {
          "order": 2,
          "software_id": "aa34795f-3401-4c28-bbe9-157a5788fd75",
          "software_name": "nginx",
          "software_version": "latest",
          "software_install_type": "package"
        }
      ],
      "errors": []
    }
  ]
}
```
</details>

### 7. Run software migration
When an array of software IDs intended for migration is provided in the migration list with identifiers such as NS ID, MCI ID, and VM ID, software migration will be performed to the target VM.

```shell
curl -X 'POST' \
  'http://127.0.0.1:8084/grasshopper/software/migrate' \
  -H 'accept: application/json' \
  -H 'Content-Type: application/json' \
  -d '{
  "software_ids": [
    "78d3664e-3eb7-4d37-bf8b-b57b7a238693",
    "aa34795f-3401-4c28-bbe9-157a5788fd75"
  ],
  "source_connection_info_id": "string",
  "target": {
    "mci_id": "mmci01",
    "namespace_id": "mig01",
    "vm_id": "rehosted-test-cm-web-1"
  }
}'
```

- Response
```json
{
  "execution_id": "aa253834-0668-4eda-8416-a0dae9f8c483",
  "execution_list": [
    {
      "order": 1,
      "software_id": "78d3664e-3eb7-4d37-bf8b-b57b7a238693",
      "software_install_type": "package",
      "software_name": "docker",
      "software_version": "latest"
    },
    {
      "order": 2,
      "software_id": "aa34795f-3401-4c28-bbe9-157a5788fd75",
      "software_install_type": "package",
      "software_name": "nginx",
      "software_version": "latest"
    }
  ]
}
```

### 8. Get software migration log
You can check the logs by providing the execution ID.

```shell
curl -X 'GET' \
  'http://127.0.0.1:8084/grasshopper/software/migrate/log/aa253834-0668-4eda-8416-a0dae9f8c483' \
  -H 'accept: application/json'
```

<details>
    <summary>Response Example</summary>

```json
{
  "uuid": "aa253834-0668-4eda-8416-a0dae9f8c483",
  "install_log": "\nPLAY [Install nginx] ***********************************************************\n\nTASK [Gathering Facts] *********************************************************\nTuesday 12 November 2024  00:48:28 +0900 (0:00:00.006)       0:00:00.006 ****** \nTuesday 12 November 2024  00:48:28 +0900 (0:00:00.006)       0:00:00.006 ****** \nok: [20.41.114.16]\n\nTASK [role : Remove deb package (Debian family)] *******************************\nTuesday 12 November 2024  00:48:30 +0900 (0:00:02.027)       0:00:02.034 ****** \nTuesday 12 November 2024  00:48:30 +0900 (0:00:02.027)       0:00:02.034 ****** \nskipping: [20.41.114.16]\n\nTASK [role : Remove rpm package (Redhat family)] *******************************\nTuesday 12 November 2024  00:48:30 +0900 (0:00:00.011)       0:00:02.046 ****** \nTuesday 12 November 2024  00:48:30 +0900 (0:00:00.011)       0:00:02.046 ****** \nskipping: [20.41.114.16]\n\nTASK [role : Install deb package (Debian family)] ******************************\nTuesday 12 November 2024  00:48:30 +0900 (0:00:00.010)       0:00:02.056 ****** \nTuesday 12 November 2024  00:48:30 +0900 (0:00:00.010)       0:00:02.056 ****** \nok: [20.41.114.16] => (item=nginx)\n\nTASK [role : Install rpm package (Redhat family)] ******************************\nTuesday 12 November 2024  00:48:35 +0900 (0:00:04.857)       0:00:06.914 ****** \nTuesday 12 November 2024  00:48:35 +0900 (0:00:04.857)       0:00:06.913 ****** \nskipping: [20.41.114.16] => (item=nginx) \nskipping: [20.41.114.16]\n\nPLAY RECAP *********************************************************************\n20.41.114.16               : ok=2    changed=0    unreachable=0    failed=0    skipped=3    rescued=0    ignored=0   \n\nTuesday 12 November 2024  00:48:35 +0900 (0:00:00.034)       0:00:06.948 ****** \n=============================================================================== \nrole : Install deb package (Debian family) ------------------------------ 4.86s\nGathering Facts --------------------------------------------------------- 2.03s\nrole : Install rpm package (Redhat family) ------------------------------ 0.03s\nrole : Remove deb package (Debian family) ------------------------------- 0.01s\nrole : Remove rpm package (Redhat family) ------------------------------- 0.01s\nTuesday 12 November 2024  00:48:35 +0900 (0:00:00.034)       0:00:06.948 ****** \n=============================================================================== \nrole -------------------------------------------------------------------- 4.91s\ngather_facts ------------------------------------------------------------ 2.03s\n~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~ \ntotal ------------------------------------------------------------------- 6.94s\n",
  "migration_log": "2024/11/12 00:48:35 [ INFO ] Starting config copier for package: nginx (UUID: aa253834-0668-4eda-8416-a0dae9f8c483)\n2024/11/12 00:48:35 [ INFO ] Finding configuration files for package: nginx\n2024/11/12 00:48:35 [ INFO ] Starting config search for package: nginx\n2024/11/12 00:48:35 [ DEBUG ] Executing config search command\n2024/11/12 00:48:40 [ DEBUG ] Processing found config files\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/init.d/nginx\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/init.d/nginx, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/init.d/nginx []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/logrotate.d/nginx\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/logrotate.d/nginx, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/logrotate.d/nginx []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/fastcgi.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/fastcgi.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/fastcgi.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/fastcgi_params\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/fastcgi_params, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/fastcgi_params []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/g gg.conf [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/g gg.conf, Status: Custom\n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/g gg.conf [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/ggg.conf [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/ggg.conf, Status: Custom\n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/ggg.conf [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/modules-enabled/50-mod-http-geoip2.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/modules-enabled/50-mod-http-geoip2.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/modules-enabled/50-mod-http-geoip2.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/modules-enabled/50-mod-http-image-filter.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/modules-enabled/50-mod-http-image-filter.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/modules-enabled/50-mod-http-image-filter.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/modules-enabled/50-mod-http-xslt-filter.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/modules-enabled/50-mod-http-xslt-filter.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/modules-enabled/50-mod-http-xslt-filter.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/modules-enabled/50-mod-mail.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/modules-enabled/50-mod-mail.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/modules-enabled/50-mod-mail.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/modules-enabled/50-mod-stream.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/modules-enabled/50-mod-stream.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/modules-enabled/50-mod-stream.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/modules-enabled/70-mod-stream-geoip2.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/modules-enabled/70-mod-stream-geoip2.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/modules-enabled/70-mod-stream-geoip2.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/nginx.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/nginx.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/nginx.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/nginx.conf.bak [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/nginx.conf.bak, Status: Custom\n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/nginx.conf.bak [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/proxy_params\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/proxy_params, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/proxy_params []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/scgi_params\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/scgi_params, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/scgi_params []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/sites-available/default [Modified]\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/sites-available/default, Status: Modified\n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/sites-available/default [Modified]\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/sites-available/default_bak [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/sites-available/default_bak, Status: Custom\n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/sites-available/default_bak [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/sites-available/ssl [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/sites-available/ssl, Status: Custom\n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/sites-available/ssl [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/sites-enabled/default [Modified]\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/sites-enabled/default, Status: Modified\n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/sites-enabled/default [Modified]\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/sites-enabled/ssl [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/sites-enabled/ssl, Status: Custom\n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/sites-enabled/ssl [Custom]\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/snippets/fastcgi-php.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/snippets/fastcgi-php.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/snippets/fastcgi-php.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/snippets/snakeoil.conf\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/snippets/snakeoil.conf, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/snippets/snakeoil.conf []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/nginx/uwsgi_params\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/nginx/uwsgi_params, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/nginx/uwsgi_params []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /etc/ufw/applications.d/nginx\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /etc/ufw/applications.d/nginx, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /etc/ufw/applications.d/nginx []\n2024/11/12 00:48:40 [ DEBUG ] Parsing config line: /usr/share/doc/nginx/copyright\n2024/11/12 00:48:40 [ DEBUG ] Parsed config - Path: /usr/share/doc/nginx/copyright, Status: \n2024/11/12 00:48:40 [ DEBUG ] Found config file: /usr/share/doc/nginx/copyright []\n2024/11/12 00:48:40 [ INFO ] Found 26 unique config files\n2024/11/12 00:48:40 [ INFO ] Found 26 configuration files for package nginx\n2024/11/12 00:48:40 [ INFO ] Starting config files copy process for 26 files\n2024/11/12 00:48:40 [ INFO ] Processing config file 1/26: /etc/init.d/nginx\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/init.d/nginx\n2024/11/12 00:48:40 [ INFO ] Processing config file 2/26: /etc/logrotate.d/nginx\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/logrotate.d/nginx\n2024/11/12 00:48:40 [ INFO ] Processing config file 3/26: /etc/nginx/fastcgi.conf\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/fastcgi.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 4/26: /etc/nginx/fastcgi_params\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/fastcgi_params\n2024/11/12 00:48:40 [ INFO ] Processing config file 5/26: /etc/nginx/g gg.conf\n2024/11/12 00:48:40 [ INFO ] Copying config file: /etc/nginx/g gg.conf [Status: Custom]\n2024/11/12 00:48:40 [ INFO ] Starting file copy process for: /etc/nginx/g gg.conf\n2024/11/12 00:48:40 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:40 [ DEBUG ] File stats - Permissions: 644, UID: 1001, GID: 1001, Type: regular empty file\n2024/11/12 00:48:40 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:40 [ INFO ] Copying file to target system\n2024/11/12 00:48:40 [ INFO ] Successfully copied file: /etc/nginx/g gg.conf\n2024/11/12 00:48:40 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/g gg.conf\n2024/11/12 00:48:40 [ INFO ] Finding certificate and key paths in file: /etc/nginx/g gg.conf\n2024/11/12 00:48:40 [ DEBUG ] Executing grep command for cert/key paths\n2024/11/12 00:48:40 [ DEBUG ] Processing grep output for pattern matches\n2024/11/12 00:48:40 [ INFO ] Found 0 potential cert/key paths\n2024/11/12 00:48:40 [ INFO ] Found 0 valid cert/key paths\n2024/11/12 00:48:40 [ DEBUG ] No cert/key files found for: /etc/nginx/g gg.conf\n2024/11/12 00:48:40 [ INFO ] Successfully processed config file: /etc/nginx/g gg.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 6/26: /etc/nginx/ggg.conf\n2024/11/12 00:48:40 [ INFO ] Copying config file: /etc/nginx/ggg.conf [Status: Custom]\n2024/11/12 00:48:40 [ INFO ] Starting file copy process for: /etc/nginx/ggg.conf\n2024/11/12 00:48:40 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:40 [ DEBUG ] File stats - Permissions: 644, UID: 1002, GID: 1002, Type: regular empty file\n2024/11/12 00:48:40 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:40 [ INFO ] Copying file to target system\n2024/11/12 00:48:40 [ INFO ] Successfully copied file: /etc/nginx/ggg.conf\n2024/11/12 00:48:40 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/ggg.conf\n2024/11/12 00:48:40 [ INFO ] Finding certificate and key paths in file: /etc/nginx/ggg.conf\n2024/11/12 00:48:40 [ DEBUG ] Executing grep command for cert/key paths\n2024/11/12 00:48:40 [ DEBUG ] Processing grep output for pattern matches\n2024/11/12 00:48:40 [ INFO ] Found 0 potential cert/key paths\n2024/11/12 00:48:40 [ INFO ] Found 0 valid cert/key paths\n2024/11/12 00:48:40 [ DEBUG ] No cert/key files found for: /etc/nginx/ggg.conf\n2024/11/12 00:48:40 [ INFO ] Successfully processed config file: /etc/nginx/ggg.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 7/26: /etc/nginx/modules-enabled/50-mod-http-geoip2.conf\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/modules-enabled/50-mod-http-geoip2.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 8/26: /etc/nginx/modules-enabled/50-mod-http-image-filter.conf\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/modules-enabled/50-mod-http-image-filter.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 9/26: /etc/nginx/modules-enabled/50-mod-http-xslt-filter.conf\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/modules-enabled/50-mod-http-xslt-filter.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 10/26: /etc/nginx/modules-enabled/50-mod-mail.conf\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/modules-enabled/50-mod-mail.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 11/26: /etc/nginx/modules-enabled/50-mod-stream.conf\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/modules-enabled/50-mod-stream.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 12/26: /etc/nginx/modules-enabled/70-mod-stream-geoip2.conf\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/modules-enabled/70-mod-stream-geoip2.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 13/26: /etc/nginx/nginx.conf\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/nginx.conf\n2024/11/12 00:48:40 [ INFO ] Processing config file 14/26: /etc/nginx/nginx.conf.bak\n2024/11/12 00:48:40 [ INFO ] Copying config file: /etc/nginx/nginx.conf.bak [Status: Custom]\n2024/11/12 00:48:40 [ INFO ] Starting file copy process for: /etc/nginx/nginx.conf.bak\n2024/11/12 00:48:40 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:40 [ DEBUG ] File stats - Permissions: 644, UID: 0, GID: 0, Type: regular file\n2024/11/12 00:48:40 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:40 [ INFO ] Copying file to target system\n2024/11/12 00:48:40 [ INFO ] Successfully copied file: /etc/nginx/nginx.conf.bak\n2024/11/12 00:48:40 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/nginx.conf.bak\n2024/11/12 00:48:40 [ INFO ] Finding certificate and key paths in file: /etc/nginx/nginx.conf.bak\n2024/11/12 00:48:40 [ DEBUG ] Executing grep command for cert/key paths\n2024/11/12 00:48:40 [ DEBUG ] Processing grep output for pattern matches\n2024/11/12 00:48:40 [ INFO ] Found 0 potential cert/key paths\n2024/11/12 00:48:40 [ INFO ] Found 0 valid cert/key paths\n2024/11/12 00:48:40 [ DEBUG ] No cert/key files found for: /etc/nginx/nginx.conf.bak\n2024/11/12 00:48:40 [ INFO ] Successfully processed config file: /etc/nginx/nginx.conf.bak\n2024/11/12 00:48:40 [ INFO ] Processing config file 15/26: /etc/nginx/proxy_params\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/proxy_params\n2024/11/12 00:48:40 [ INFO ] Processing config file 16/26: /etc/nginx/scgi_params\n2024/11/12 00:48:40 [ DEBUG ] Skipping unmodified config: /etc/nginx/scgi_params\n2024/11/12 00:48:40 [ INFO ] Processing config file 17/26: /etc/nginx/sites-available/default\n2024/11/12 00:48:40 [ INFO ] Copying config file: /etc/nginx/sites-available/default [Status: Modified]\n2024/11/12 00:48:40 [ INFO ] Starting file copy process for: /etc/nginx/sites-available/default\n2024/11/12 00:48:40 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:40 [ DEBUG ] File stats - Permissions: 644, UID: 0, GID: 0, Type: regular file\n2024/11/12 00:48:40 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:40 [ INFO ] Copying file to target system\n2024/11/12 00:48:41 [ INFO ] Successfully copied file: /etc/nginx/sites-available/default\n2024/11/12 00:48:41 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/sites-available/default\n2024/11/12 00:48:41 [ INFO ] Finding certificate and key paths in file: /etc/nginx/sites-available/default\n2024/11/12 00:48:41 [ DEBUG ] Executing grep command for cert/key paths\n2024/11/12 00:48:41 [ DEBUG ] Processing grep output for pattern matches\n2024/11/12 00:48:41 [ INFO ] Found 0 potential cert/key paths\n2024/11/12 00:48:41 [ INFO ] Found 0 valid cert/key paths\n2024/11/12 00:48:41 [ DEBUG ] No cert/key files found for: /etc/nginx/sites-available/default\n2024/11/12 00:48:41 [ INFO ] Successfully processed config file: /etc/nginx/sites-available/default\n2024/11/12 00:48:41 [ INFO ] Processing config file 18/26: /etc/nginx/sites-available/default_bak\n2024/11/12 00:48:41 [ INFO ] Copying config file: /etc/nginx/sites-available/default_bak [Status: Custom]\n2024/11/12 00:48:41 [ INFO ] Starting file copy process for: /etc/nginx/sites-available/default_bak\n2024/11/12 00:48:41 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:41 [ DEBUG ] File stats - Permissions: 644, UID: 0, GID: 0, Type: regular file\n2024/11/12 00:48:41 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:41 [ INFO ] Copying file to target system\n2024/11/12 00:48:41 [ INFO ] Successfully copied file: /etc/nginx/sites-available/default_bak\n2024/11/12 00:48:41 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/sites-available/default_bak\n2024/11/12 00:48:41 [ INFO ] Finding certificate and key paths in file: /etc/nginx/sites-available/default_bak\n2024/11/12 00:48:41 [ DEBUG ] Executing grep command for cert/key paths\n2024/11/12 00:48:41 [ DEBUG ] Processing grep output for pattern matches\n2024/11/12 00:48:41 [ INFO ] Found 0 potential cert/key paths\n2024/11/12 00:48:41 [ INFO ] Found 0 valid cert/key paths\n2024/11/12 00:48:41 [ DEBUG ] No cert/key files found for: /etc/nginx/sites-available/default_bak\n2024/11/12 00:48:41 [ INFO ] Successfully processed config file: /etc/nginx/sites-available/default_bak\n2024/11/12 00:48:41 [ INFO ] Processing config file 19/26: /etc/nginx/sites-available/ssl\n2024/11/12 00:48:41 [ INFO ] Copying config file: /etc/nginx/sites-available/ssl [Status: Custom]\n2024/11/12 00:48:41 [ INFO ] Starting file copy process for: /etc/nginx/sites-available/ssl\n2024/11/12 00:48:41 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:41 [ DEBUG ] File stats - Permissions: 644, UID: 0, GID: 0, Type: regular file\n2024/11/12 00:48:41 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:41 [ INFO ] Copying file to target system\n2024/11/12 00:48:41 [ INFO ] Successfully copied file: /etc/nginx/sites-available/ssl\n2024/11/12 00:48:41 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/sites-available/ssl\n2024/11/12 00:48:41 [ INFO ] Finding certificate and key paths in file: /etc/nginx/sites-available/ssl\n2024/11/12 00:48:41 [ DEBUG ] Executing grep command for cert/key paths\n2024/11/12 00:48:41 [ DEBUG ] Processing grep output for pattern matches\n2024/11/12 00:48:41 [ DEBUG ] Found unique cert/key path: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:41 [ DEBUG ] Found unique cert/key path: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:41 [ INFO ] Found 2 potential cert/key paths\n2024/11/12 00:48:41 [ DEBUG ] Verifying path existence: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:41 [ INFO ] Verified valid cert/key path: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:41 [ DEBUG ] Verifying path existence: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:41 [ INFO ] Verified valid cert/key path: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:41 [ INFO ] Found 2 valid cert/key paths\n2024/11/12 00:48:41 [ INFO ] Found 2 cert/key files for /etc/nginx/sites-available/ssl\n2024/11/12 00:48:41 [ INFO ] Copying cert/key file 1/2: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:41 [ INFO ] Starting file copy process for: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:41 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:41 [ DEBUG ] File stats - Permissions: 644, UID: 0, GID: 0, Type: regular file\n2024/11/12 00:48:41 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:41 [ INFO ] Copying file to target system\n2024/11/12 00:48:41 [ INFO ] Successfully copied file: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:41 [ INFO ] Successfully copied cert/key file: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:41 [ INFO ] Copying cert/key file 2/2: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:41 [ INFO ] Starting file copy process for: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:41 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:41 [ DEBUG ] File stats - Permissions: 600, UID: 0, GID: 0, Type: regular file\n2024/11/12 00:48:41 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:41 [ INFO ] Copying file to target system\n2024/11/12 00:48:41 [ INFO ] Successfully copied file: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:41 [ INFO ] Successfully copied cert/key file: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:41 [ INFO ] Successfully processed config file: /etc/nginx/sites-available/ssl\n2024/11/12 00:48:41 [ INFO ] Processing config file 20/26: /etc/nginx/sites-enabled/default\n2024/11/12 00:48:41 [ INFO ] Copying config file: /etc/nginx/sites-enabled/default [Status: Modified]\n2024/11/12 00:48:41 [ INFO ] Starting file copy process for: /etc/nginx/sites-enabled/default\n2024/11/12 00:48:41 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:41 [ DEBUG ] File stats - Permissions: 777, UID: 0, GID: 0, Type: symbolic link\n2024/11/12 00:48:41 [ INFO ] File is a symbolic link\n2024/11/12 00:48:41 [ DEBUG ] Symlink target: /etc/nginx/sites-available/default\n2024/11/12 00:48:41 [ INFO ] Copying file to target system\n2024/11/12 00:48:41 [ DEBUG ] Creating symlink on target system\n2024/11/12 00:48:41 [ INFO ] Successfully copied file: /etc/nginx/sites-enabled/default\n2024/11/12 00:48:41 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/sites-enabled/default\n2024/11/12 00:48:41 [ INFO ] Finding certificate and key paths in file: /etc/nginx/sites-enabled/default\n2024/11/12 00:48:41 [ DEBUG ] Executing grep command for cert/key paths\n2024/11/12 00:48:41 [ DEBUG ] Processing grep output for pattern matches\n2024/11/12 00:48:41 [ INFO ] Found 0 potential cert/key paths\n2024/11/12 00:48:41 [ INFO ] Found 0 valid cert/key paths\n2024/11/12 00:48:41 [ DEBUG ] No cert/key files found for: /etc/nginx/sites-enabled/default\n2024/11/12 00:48:41 [ INFO ] Successfully processed config file: /etc/nginx/sites-enabled/default\n2024/11/12 00:48:41 [ INFO ] Processing config file 21/26: /etc/nginx/sites-enabled/ssl\n2024/11/12 00:48:41 [ INFO ] Copying config file: /etc/nginx/sites-enabled/ssl [Status: Custom]\n2024/11/12 00:48:41 [ INFO ] Starting file copy process for: /etc/nginx/sites-enabled/ssl\n2024/11/12 00:48:41 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:41 [ DEBUG ] File stats - Permissions: 777, UID: 0, GID: 0, Type: symbolic link\n2024/11/12 00:48:41 [ INFO ] File is a symbolic link\n2024/11/12 00:48:41 [ DEBUG ] Symlink target: /etc/nginx/sites-available/ssl\n2024/11/12 00:48:42 [ INFO ] Copying file to target system\n2024/11/12 00:48:42 [ DEBUG ] Creating symlink on target system\n2024/11/12 00:48:42 [ INFO ] Successfully copied file: /etc/nginx/sites-enabled/ssl\n2024/11/12 00:48:42 [ DEBUG ] Searching for associated cert/key files for: /etc/nginx/sites-enabled/ssl\n2024/11/12 00:48:42 [ INFO ] Finding certificate and key paths in file: /etc/nginx/sites-enabled/ssl\n2024/11/12 00:48:42 [ DEBUG ] Executing grep command for cert/key paths\n2024/11/12 00:48:42 [ DEBUG ] Processing grep output for pattern matches\n2024/11/12 00:48:42 [ DEBUG ] Found unique cert/key path: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:42 [ DEBUG ] Found unique cert/key path: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:42 [ INFO ] Found 2 potential cert/key paths\n2024/11/12 00:48:42 [ DEBUG ] Verifying path existence: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:42 [ INFO ] Verified valid cert/key path: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:42 [ DEBUG ] Verifying path existence: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:42 [ INFO ] Verified valid cert/key path: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:42 [ INFO ] Found 2 valid cert/key paths\n2024/11/12 00:48:42 [ INFO ] Found 2 cert/key files for /etc/nginx/sites-enabled/ssl\n2024/11/12 00:48:42 [ INFO ] Copying cert/key file 1/2: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:42 [ INFO ] Starting file copy process for: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:42 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:42 [ DEBUG ] File stats - Permissions: 644, UID: 0, GID: 0, Type: regular file\n2024/11/12 00:48:42 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:42 [ INFO ] Copying file to target system\n2024/11/12 00:48:42 [ INFO ] Successfully copied file: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:42 [ INFO ] Successfully copied cert/key file: /etc/nginx/certs/innogrid.com.crt\n2024/11/12 00:48:42 [ INFO ] Copying cert/key file 2/2: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:42 [ INFO ] Starting file copy process for: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:42 [ DEBUG ] Getting file statistics\n2024/11/12 00:48:42 [ DEBUG ] File stats - Permissions: 600, UID: 0, GID: 0, Type: regular file\n2024/11/12 00:48:42 [ DEBUG ] Reading regular file content\n2024/11/12 00:48:42 [ INFO ] Copying file to target system\n2024/11/12 00:48:42 [ INFO ] Successfully copied file: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:42 [ INFO ] Successfully copied cert/key file: /etc/nginx/certs/innogrid.com.key\n2024/11/12 00:48:42 [ INFO ] Successfully processed config file: /etc/nginx/sites-enabled/ssl\n2024/11/12 00:48:42 [ INFO ] Processing config file 22/26: /etc/nginx/snippets/fastcgi-php.conf\n2024/11/12 00:48:42 [ DEBUG ] Skipping unmodified config: /etc/nginx/snippets/fastcgi-php.conf\n2024/11/12 00:48:42 [ INFO ] Processing config file 23/26: /etc/nginx/snippets/snakeoil.conf\n2024/11/12 00:48:42 [ DEBUG ] Skipping unmodified config: /etc/nginx/snippets/snakeoil.conf\n2024/11/12 00:48:42 [ INFO ] Processing config file 24/26: /etc/nginx/uwsgi_params\n2024/11/12 00:48:42 [ DEBUG ] Skipping unmodified config: /etc/nginx/uwsgi_params\n2024/11/12 00:48:42 [ INFO ] Processing config file 25/26: /etc/ufw/applications.d/nginx\n2024/11/12 00:48:42 [ DEBUG ] Skipping unmodified config: /etc/ufw/applications.d/nginx\n2024/11/12 00:48:42 [ INFO ] Processing config file 26/26: /usr/share/doc/nginx/copyright\n2024/11/12 00:48:42 [ DEBUG ] Skipping unmodified config: /usr/share/doc/nginx/copyright\n2024/11/12 00:48:42 [ INFO ] Completed config files copy process\n2024/11/12 00:48:42 [ INFO ] Successfully completed config copy process for package: nginx\n2024/11/12 00:48:42 [ INFO ] Starting service migration for package: nginx (UUID: aa253834-0668-4eda-8416-a0dae9f8c483)\n2024/11/12 00:48:42 [ DEBUG ] Detecting source system type\n2024/11/12 00:48:42 [ INFO ] Detected system ID: ubuntu\n2024/11/12 00:48:42 [ DEBUG ] Detecting target system type\n2024/11/12 00:48:42 [ INFO ] Detected system ID: ubuntu\n2024/11/12 00:48:42 [ INFO ] Finding services for package: nginx\n2024/11/12 00:48:42 [ INFO ] Found 1 relevant services for package\n2024/11/12 00:48:42 [ INFO ] Getting service info for: nginx\n2024/11/12 00:48:42 [ DEBUG ] Service nginx active status: true\n2024/11/12 00:48:42 [ DEBUG ] Service nginx state: enabled\n2024/11/12 00:48:42 [ DEBUG ] Service nginx enabled status: true\n2024/11/12 00:48:42 [ INFO ] Completed service info gathering for nginx\n2024/11/12 00:48:42 [ INFO ] Completed service discovery with 1 services\n2024/11/12 00:48:42 [ INFO ] Stopping services in dependency order\n2024/11/12 00:48:42 [ INFO ] Stopping service: nginx\n2024/11/12 00:48:42 [ INFO ] Setting service enable/disable states\n2024/11/12 00:48:42 [ INFO ] Enabling service: nginx\n2024/11/12 00:48:43 [ INFO ] Starting services in reverse dependency order\n2024/11/12 00:48:43 [ INFO ] Starting service: nginx\n2024/11/12 00:48:43 [ INFO ] Successfully started service: nginx\n2024/11/12 00:48:43 [ INFO ] Verifying final states\n2024/11/12 00:48:43 [ INFO ] Starting service migration for package: nginx\n2024/11/12 00:48:43 [ DEBUG ] Detecting source PID\n2024/11/12 00:48:43 [ DEBUG ] Source PID: 408710\n2024/11/12 00:48:43 [ DEBUG ] Detecting target PID\n2024/11/12 00:48:43 [ DEBUG ] Target PID: 12630\n2024/11/12 00:48:43 [ DEBUG ] Retrieving source listening connections\n"
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
