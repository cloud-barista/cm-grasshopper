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
  * Ubuntu 23.10, Ubuntu 22.04, Ubuntu 18.04, Rocky Linux 9
* Language:
  * Go: 1.21.5

## How to run

1. Build the binary
     ```shell
     make
     ```

2. Write the configuration file.
    - Configuration file name is 'cm-grasshopper.yaml'
    - The configuration file must be placed in one of the following directories.
        - .cm-grasshopper/conf directory under user's home directory
        - 'conf' directory where running the binary
        - 'conf' directory where placed in the path of 'CMGRASSHOPPER_ROOT' environment variable
    - Configuration options
        - listen
            - port : Listen port of the API.
    - Configuration file example
      ```yaml
      cm-grasshopper:
          listen:
              port: 8084
      ```

3. Run with privileges
     ```shell
     sudo ./cm-grasshopper
     ```
#### Download source code

Clone CM-Grasshopper repository

```bash
git clone https://github.com/cloud-barista/cm-grasshopper.git ${HOME}/cm-grasshopper
```

#### Build CM-Grasshopper

Build CM-Grasshopper source code

```bash
cd ${HOME}/cm-grasshopper
make build
```

(Optional) Update Swagger API document
```bash
cd ${HOME}/cm-grasshopper
make swag
```

Access to Swagger UI
(Default link) http://localhost:8056/Grasshopper/swagger/index.html

#### Run CM-Grasshopper binary

Run CM-Grasshopper server

```bash
cd ${HOME}/cm-grasshopper
make build
./cm-grasshopper
```