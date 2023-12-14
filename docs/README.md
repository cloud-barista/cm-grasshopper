# Documentation

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
