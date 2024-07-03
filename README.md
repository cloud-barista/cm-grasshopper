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
  * Go: 1.21.6

## How to run

1. Write the configuration file.
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

2. Build and run the binary
 ```shell
 make run
 ```

### Health-check

Check if CM-Grasshopper is running

```bash
curl http://localhost:8084/grasshopper/readyz

# Output if it's running successfully
# {"message":"CM-Grasshopper API server is ready"}
```

## Check out all APIs
* [Grasshopper APIs (Swagger Document)](https://cloud-barista.github.io/cb-tumblebug-api-web/?url=https://raw.githubusercontent.com/cloud-barista/cm-grasshopper/main/pkg/api/rest/docs/swagger.yaml)
