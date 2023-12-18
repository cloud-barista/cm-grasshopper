# Management of software migration
This is a sub-system on Cloud-Barista platform provides a features of management software migration for cloud migration.

## Overview

* Collect information of installed software.
* Collect configuration files of installed software.
* Migrate software and configure for target cloud platform.

## Execution and development environment
* Tested operating systems (OSs):
    * Ubuntu 23.10, Ubuntu 22.04, Ubuntu 18.04
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
