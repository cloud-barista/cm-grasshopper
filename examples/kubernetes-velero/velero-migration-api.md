# Velero Migration API Examples

## Base URL

```text
http://localhost:8084/grasshopper
```



## HTTP files

- httpYac environment: [http-client.env.json](/Users/taking/Documents/innogrid/projects/cm-grasshopper/examples/kubernetes-velero/api/http-client.env.json)
- [01-install.http](/Users/taking/Documents/innogrid/projects/cm-grasshopper/examples/kubernetes-velero/api/01-install.http)
- [02-health.http](/Users/taking/Documents/innogrid/projects/cm-grasshopper/examples/kubernetes-velero/api/02-health.http)
- [03-precheck.http](/Users/taking/Documents/innogrid/projects/cm-grasshopper/examples/kubernetes-velero/api/03-precheck.http)
- [04-backup.http](/Users/taking/Documents/innogrid/projects/cm-grasshopper/examples/kubernetes-velero/api/04-backup.http)
- [05-restore.http](/Users/taking/Documents/innogrid/projects/cm-grasshopper/examples/kubernetes-velero/api/05-restore.http)
- [06-execute.http](/Users/taking/Documents/innogrid/projects/cm-grasshopper/examples/kubernetes-velero/api/06-execute.http)
- [07-job.http](/Users/taking/Documents/innogrid/projects/cm-grasshopper/examples/kubernetes-velero/api/07-job.http)



## API list

| Category | Method | Path | Description |
|---|---|---|---|
| Velero | `POST` | `/velero/{role}/install` | source 또는 target Velero 설치/업그레이드 |
| Velero | `POST` | `/velero/{role}/health` | source 또는 target Velero 상태 확인 |
| Source backup | `POST` | `/velero/source/backups` | source backup 생성 |
| Source backup | `POST` | `/velero/source/backups/list` | source backup 목록 조회 |
| Source backup | `POST` | `/velero/source/backups/{name}` | source backup 단건 조회 |
| Source backup | `POST` | `/velero/source/backups/{name}/validate` | source backup 상태 검증 |
| Source backup | `POST` | `/velero/source/backups/{name}/delete` | source backup 삭제 |
| Target restore | `POST` | `/velero/target/restores` | target restore 생성 |
| Target restore | `POST` | `/velero/target/restores/list` | target restore 목록 조회 |
| Target restore | `POST` | `/velero/target/restores/{name}` | target restore 단건 조회 |
| Target restore | `POST` | `/velero/target/restores/{name}/validate` | target restore 상태 검증 |
| Target restore | `POST` | `/velero/target/restores/{name}/delete` | target restore 삭제 |
| Migration | `POST` | `/velero/migration/precheck` | migration 실행 전 점검 |
| Migration | `POST` | `/velero/migration/execute` | source backup + target restore 비동기 실행 |
| Job | `GET` | `/job/status/{jobId}` | job 단건 상태 조회 |
| Job | `GET` | `/job/log/{jobId}` | job 로그 조회 |
| Job | `GET` | `/job/status` | job 목록 조회 |



## Common variables

httpYac 기준으로는 각 `.http` 파일 맨 위에 변수를 반복 선언하지 않고, 아래 환경 파일 하나로 관리합니다.

```json
{
  "local": {
    "baseUrl": "http://localhost:8084/grasshopper",
    "base64_source_kubeconfig": "CHANGE_ME",
    "base64_target_kubeconfig": "CHANGE_ME",
    "minio_url": "minio.example.com",
    "minio_accesskey": "CHANGE_ME",
    "minio_secretkey": "CHANGE_ME",
    "minio_bucket": "velero",
    "job_id": "CHANGE_ME"
  }
}
```



## Recommended defaults

- `filesystem + NFS` 테스트가 가장 안정적입니다.
- `precheck` 기본 응답은 compact 요약형이고, 상세가 필요하면 `?verbose=true`를 사용합니다.
- source PVC가 이미 `nfs-client`라면 restore 시 `storageClassMappings`는 생략해도 됩니다.
- FSB 테스트에서는 backup `includedResources`를 과도하게 줄이지 않는 편이 안전합니다.
- 특히 PVC를 포함한다면 `persistentvolumes`도 같이 포함하거나, 가장 단순하게는 `includedResources`를 생략하는 것을 권장합니다.
- restore는 target cluster가 backup sync를 끝낸 뒤 실행해야 합니다.

## Restore guardrails

- 수동 restore 생성 전에 target cluster에서 해당 backup이 실제로 조회되는지 먼저 확인합니다.
- backup sync 전에 `POST /velero/target/restores`를 호출하면 API는 400과 함께 retry 안내 메시지를 반환합니다.
- restore 조회/검증 응답에는 `validationErrors`가 포함될 수 있으니, `phase`와 함께 같이 확인하는 것이 좋습니다.



## Filesystem + NFS 기준 권장 흐름

1. source install
2. target install
3. source health
4. target health
5. migration precheck
6. source backup create
7. target cluster에서 backup sync 확인
8. target restore create
9. restore validate / PVC / Pod / 파일 데이터 확인



## 1. Install

source

```bash
curl -X POST http://localhost:8084/grasshopper/velero/source/install \
  -H 'Content-Type: application/json' \
  -d '{
    "sourceCluster": {
      "name": "source-cluster",
      "namespace": "velero",
      "kubeconfig": "{{base64_source_kubeconfig}}"
    },
    "storage": {
      "minio": {
        "endpoint": "{{minio_url}}",
        "accessKey": "{{minio_accesskey}}",
        "secretKey": "{{minio_secretkey}}",
        "bucket": "{{minio_bucket}}",
        "useSSL": true
      }
    },
    "install": {
      "force": false,
      "volumeBackupMode": "filesystem"
    }
  }'
```

target

```bash
curl -X POST http://localhost:8084/grasshopper/velero/target/install \
  -H 'Content-Type: application/json' \
  -d '{
    "targetCluster": {
      "name": "target-cluster",
      "namespace": "velero",
      "kubeconfig": "{{base64_target_kubeconfig}}"
    },
    "storage": {
      "minio": {
        "endpoint": "{{minio_url}}",
        "accessKey": "{{minio_accesskey}}",
        "secretKey": "{{minio_secretkey}}",
        "bucket": "{{minio_bucket}}",
        "useSSL": true
      }
    },
    "install": {
      "force": false,
      "volumeBackupMode": "filesystem"
    }
  }'
```



## 2. Health

source

```bash
curl -X POST http://localhost:8084/grasshopper/velero/source/health \
  -H 'Content-Type: application/json' \
  -d '{
    "sourceCluster": {
      "name": "source-cluster",
      "namespace": "velero",
      "kubeconfig": "{{base64_source_kubeconfig}}"
    }
  }'
```

target

```bash
curl -X POST http://localhost:8084/grasshopper/velero/target/health \
  -H 'Content-Type: application/json' \
  -d '{
    "targetCluster": {
      "name": "target-cluster",
      "namespace": "velero",
      "kubeconfig": "{{base64_target_kubeconfig}}"
    }
  }'
```



## 3. Precheck

compact

```bash
curl -X POST 'http://localhost:8084/grasshopper/velero/migration/precheck?verbose=false' \
  -H 'Content-Type: application/json' \
  -d '{
    "sourceCluster": {
      "name": "source-cluster",
      "namespace": "velero",
      "kubeconfig": "{{base64_source_kubeconfig}}"
    },
    "targetCluster": {
      "name": "target-cluster",
      "namespace": "velero",
      "kubeconfig": "{{base64_target_kubeconfig}}"
    },
    "storage": {
      "minio": {
        "endpoint": "{{minio_url}}",
        "accessKey": "{{minio_accesskey}}",
        "secretKey": "{{minio_secretkey}}",
        "bucket": "{{minio_bucket}}",
        "useSSL": true
      }
    },
    "precheck": {
      "sourceNamespace": "demo",
      "targetNamespace": "demo-restored",
      "storageClassMappings": {
        "nfs-client": "nfs-client"
      }
    }
  }'
```

상세

```bash
curl -X POST 'http://localhost:8084/grasshopper/velero/migration/precheck?verbose=true' \
  -H 'Content-Type: application/json' \
  -d '{ ... }'
```



## 4. Source backup create

filesystem + NFS 예시

```bash
curl -X POST http://localhost:8084/grasshopper/velero/source/backups \
  -H 'Content-Type: application/json' \
  -d '{
    "sourceCluster": {
      "name": "source-cluster",
      "namespace": "velero",
      "kubeconfig": "{{base64_source_kubeconfig}}"
    },
    "backup": {
      "name": "backup-demo-fsb",
      "sourceNamespace": "demo",
      "volumeBackupMode": "filesystem",
      "includedResources": [
        "deployments",
        "services",
        "configmaps",
        "secrets",
        "persistentvolumes",
        "persistentvolumeclaims",
        "pods"
      ]
    }
  }'
```

권장

- 새 테스트마다 새 `backup.name` 사용
- target restore 전에 target cluster에서 backup sync 여부 확인

확인

```bash
velero backup describe backup-demo-fsb --kubeconfig /tmp/source-kubeconfig
velero backup get --kubeconfig /tmp/target-kubeconfig
```



## 5. Target restore create

restore 전에 target cluster에서 backup sync를 먼저 확인합니다.

```bash
curl -X POST http://localhost:8084/grasshopper/velero/source/backups/backup-demo-fsb \
  -H 'Content-Type: application/json' \
  -d '{
    "sourceCluster": {
      "name": "target-cluster",
      "namespace": "velero",
      "kubeconfig": "{{base64_target_kubeconfig}}"
    }
  }'
```

```bash
curl -X POST http://localhost:8084/grasshopper/velero/target/restores \
  -H 'Content-Type: application/json' \
  -d '{
    "targetCluster": {
      "name": "target-cluster",
      "namespace": "velero",
      "kubeconfig": "{{base64_target_kubeconfig}}"
    },
    "restore": {
      "name": "restore-demo-fsb",
      "backupName": "backup-demo-fsb",
      "sourceNamespace": "demo",
      "targetNamespace": "demo-restored",
      "storageClassMappings": {
        "nfs-client": "nfs-client"
      },
      "existingResourcePolicy": "update",
      "restorePVs": true
    }
  }'
```

주의

- target cluster에서 `backup-demo-fsb`가 보이기 전에 restore를 요청하면 API가 retry 안내와 함께 실패합니다.
- 이전 테스트의 PVC/PV가 남아 있으면 결과가 오염될 수 있으니, 새 `restore.name`과 새 target namespace 사용을 권장합니다.



## 6. Migration execute

```bash
curl -X POST http://localhost:8084/grasshopper/velero/migration/execute \
  -H 'Content-Type: application/json' \
  -d '{
    "sourceCluster": {
      "name": "source-cluster",
      "namespace": "velero",
      "kubeconfig": "{{base64_source_kubeconfig}}"
    },
    "targetCluster": {
      "name": "target-cluster",
      "namespace": "velero",
      "kubeconfig": "{{base64_target_kubeconfig}}"
    },
    "storage": {
      "minio": {
        "endpoint": "{{minio_url}}",
        "accessKey": "{{minio_accesskey}}",
        "secretKey": "{{minio_secretkey}}",
        "bucket": "{{minio_bucket}}",
        "useSSL": true
      }
    },
    "migration": {
      "backupName": "backup-demo-fsb",
      "restoreName": "restore-demo-fsb",
      "sourceNamespace": "demo",
      "targetNamespace": "demo-restored",
      "volumeBackupMode": "filesystem",
      "includedResources": [
        "deployments",
        "services",
        "configmaps",
        "secrets",
        "persistentvolumes",
        "persistentvolumeclaims",
        "pods"
      ],
      "storageClassMappings": {
        "nfs-client": "nfs-client"
      },
      "existingResourcePolicy": "update",
      "restorePVs": true
    }
  }'
```



## 7. Job

```bash
curl http://localhost:8084/grasshopper/job/status/{{job_id}}
```

- `current_stage` 로 현재 상위 단계를 빠르게 볼 수 있습니다.
- 예: `precheck`, `backup`, `backup_sync`, `restore`, `completed`

```bash
curl http://localhost:8084/grasshopper/job/log/{{job_id}}
```



## Notes

- `kind + NFS`는 `filesystem` 테스트용으로 보는 것이 맞습니다.
- `kind + csi-hostpath`는 snapshot 동작 확인에는 쓸 수 있어도 cross-cluster snapshot restore 검증에는 적합하지 않습니다.
- restore 후 PVC가 `Pending`이면 target storage class, target provisioner, 기존 PVC/PV 잔존 여부를 먼저 확인하세요.

## Troubleshooting

- restore create가 바로 실패하고 `backup is not available on target cluster yet`가 보이면:
  - target cluster에 backup sync가 아직 끝나지 않은 상태입니다.
  - target kubeconfig 기준으로 `POST /velero/source/backups/{name}`를 다시 호출해 backup 조회부터 확인하세요.
- restore 응답의 `phase`가 `FailedValidation`이면:
  - `validationErrors` 필드를 먼저 확인하세요.
  - 가장 흔한 원인은 backup name 오타 또는 target cluster backup sync 지연입니다.
- restore 후 PVC가 `Pending`이면:
  - target storage class, provisioner, 기존 PVC 잔존 여부를 먼저 확인하세요.
