# API 가이드


## API 작성 규칙
[cm-beetle Useful samples to add new APIs 참고](https://github.com/cloud-barista/cm-beetle/blob/main/docs/useful-samples-to-add-new-apis.md)

## Using APIs

### Register target
- URL : http://127.0.0.1:8084/target/register
- Method : POST
- Parameters
  - Needed
    - honeybee_address : {IP or Domain}:{Port}
- Example
    ```
    http://127.0.0.1:8084/target/register?honeybee_address=127.0.0.1:8082
    ```

### Get target
- URL : http://127.0.0.1:8084/target/get
- Method : GET
- Parameters
  - Needed
    - uuid : xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx
- Example
  ```
  http://127.0.0.1:8084/target/get?uuid=9e7cbea7-2ff1-41df-9916-ebea29c27a07
  ```

### List targets
- URL : http://127.0.0.1:8084/target/list
- Method : GET
- Parameters
  - Options
    - uuid : Any string to match
    - honeybee_address : Any string to match
- Example
    ```
    http://127.0.0.1:8084/target/list?uuid=9e&honeybee_address=8082
    ```

### Update target
- URL : http://127.0.0.1:8084/target/update
- Method : POST
- Parameters
  - Needed
    - uuid : xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx
    - honeybee_address : {IP or Domain}:{Port}
- Example
    ```
    http://127.0.0.1:8084/target/update?uuid=3cbfdf42-047f-45b8-8d93-2cea7201ea59&honeybee_address=:::8082
    ```

### Delete target
- URL : http://127.0.0.1:8084/target/list
- Method : GET
- Parameters
  - Needed
    - uuid : xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx
- Example
    ```
    http://127.0.0.1:8084/target/delete?uuid=18f9fd22-395d-4dd1-b70b-1c59afebce11
    ```
