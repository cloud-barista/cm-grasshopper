# Velero Migration 테스트 방법

이 문서는 `cm-grasshopper`에 추가한 Kubernetes Velero migration 기능을 실제로 테스트하는 방법을 한글로 단계별 정리한 문서입니다.



테스트 대상 기능은 아래입니다.

- source cluster -> target cluster 백업/복구
- FSB 기반 PVC 데이터 백업/복구
- namespace remap restore
- StorageClass mapping
- existing resource policy



관련 예제 파일

- [README.md](README.md)
- [source-demo-app.yaml](shared/source-demo-app.yaml)
- [velero-migration-api.md](velero-migration-api.md)
- [http-client.env.json](api/http-client.env.json)
- [01-install.http](api/01-install.http)
- [02-health.http](api/02-health.http)
- [03-precheck.http](api/03-precheck.http)
- [04-backup.http](api/04-backup.http)
- [05-restore.http](api/05-restore.http)
- [06-execute.http](api/06-execute.http)
- [07-job.http](api/07-job.http)
- [write-test-data.sh](shared/write-test-data.sh)
- [verify-restored-data.sh](shared/verify-restored-data.sh)



## 1. 준비물

먼저 아래 항목이 준비되어 있어야 합니다.

1. source Kubernetes cluster
2. target Kubernetes cluster
3. source kubeconfig
4. target kubeconfig
5. MinIO endpoint, accessKey, secretKey
6. source cluster에서 사용할 StorageClass 이름
7. target cluster에서 사용할 StorageClass 이름
8. 실행 중인 `cm-grasshopper`



예시

- source storage class: `old-sc`
- target storage class: `new-sc`
- source namespace: `demo`
- target namespace: `demo-restored`



## 2. kubeconfig 준비

`cm-grasshopper` API는 kubeconfig 원문이 아니라 base64 인코딩된 값을 받습니다.



예시

```bash
base64 -i ~/.kube/source-config | tr -d '\n'
```

```bash
base64 -i ~/.kube/target-config | tr -d '\n'
```



주의

- 개행 없이 한 줄로 넣는 것이 편합니다.
- `.http` 파일이나 curl 예시에 넣을 때 `{{base64_source_kubeconfig}}`, `{{base64_target_kubeconfig}}` 자리에 치환합니다.



## 3. source cluster에 테스트 앱 배포

먼저 source 쪽에 백업할 테스트 앱을 올립니다.



사용 파일

- [source-demo-app.yaml](shared/source-demo-app.yaml)



이 파일에는 아래가 들어 있습니다.

- `demo` namespace
- `demo-data` PVC
- `app` Deployment
- `app` Service
- `app-config` ConfigMap
- `app-secret` Secret



### 3-1. StorageClass 이름 수정

먼저 YAML 안의 `storageClassName: old-sc`를 source cluster에 실제 존재하는 StorageClass 이름으로 바꿉니다.



예시

```yaml
storageClassName: nfs-client
```



### 3-2. 배포

```bash
kubectl apply -f examples/kubernetes-velero/shared/source-demo-app.yaml
```



### 3-3. 배포 확인

```bash
kubectl -n demo get all
kubectl -n demo get pvc
```



정상 기대값

- `app` pod가 Running
- `demo-data` PVC가 Bound



## 4. source PVC에 테스트 데이터 쓰기

FSB 복구가 실제로 되는지 확인하려면 PVC 안에 파일을 하나 써두는 게 좋습니다.



사용 스크립트

- [write-test-data.sh](shared/write-test-data.sh)



실행

```bash
bash examples/kubernetes-velero/shared/write-test-data.sh
```



기본 동작

- namespace: `demo`
- deployment: `app`
- file path: `/usr/share/nginx/html/data/check.txt`
- 값: `hello-velero`



직접 확인

```bash
kubectl -n demo exec deploy/app -- sh -c 'cat /usr/share/nginx/html/data/check.txt'
```



## 5. API 호출 방식 선택

테스트는 두 가지 방식 중 하나로 진행하면 됩니다.

1. `curl` 사용
   - [velero-migration-api.md](velero-migration-api.md)
2. REST Client 사용
   - flow files: `01-` ~ `07-` `.http`
   - environment file: `api/http-client.env.json`

처음엔 `.http` 파일로 하는 것을 권장합니다. source/target kubeconfig와 MinIO 값만 바꾸면 순서대로 테스트하기 편합니다.



## 6. source/target Velero 설치

먼저 source cluster와 target cluster 모두에 Velero가 설치되어야 합니다.



실행 순서

1. `Source install`
2. `Target install`



API

- `POST /grasshopper/velero/source/install`
- `POST /grasshopper/velero/target/install`

응답에는 `job_id`가 반환됩니다.



### 6-1. job 상태 확인

```bash
curl http://localhost:8084/grasshopper/job/status/{{job_id}}
```

- `current_stage` 를 보면 지금 precheck/backup/backup_sync/restore 중 어디인지 빠르게 파악할 수 있습니다.

```bash
curl http://localhost:8084/grasshopper/job/log/{{job_id}}
```



정상 기대값

- status가 `completed`
- log에 Velero install 완료 메시지 존재



## 7. source/target health 확인

Velero 설치가 끝나면 health 체크를 합니다.



실행 순서

1. `Source health`
2. `Target health`



API

- `POST /grasshopper/velero/source/health`
- `POST /grasshopper/velero/target/health`



정상 기대값

```json
{
  "namespace": "velero",
  "status": "ok"
}
```



## 8. migration precheck 실행

백업/복구 전에 source, target, MinIO가 모두 접근 가능한지 확인합니다.



API

- `POST /grasshopper/velero/migration/precheck`



정상 기대값

- `status: ready` 또는 `ready_with_warnings`
- source 정보 존재
- target 정보 존재
- storage endpoint/bucket 정보 존재
- 필요 시 `warnings` 배열 존재
- 실행 불가한 경우 `status: not_ready`, `errors` 배열 존재

이 단계에서 실패하면 실제 migration을 진행하지 말고 먼저 원인을 확인하는 것이 좋습니다.

precheck에서 주로 확인하는 항목은 아래입니다.

- source namespace가 실제로 존재하는지
- target namespace가 이미 있는지, 없으면 restore 시 생성 예정인지
- source PVC가 사용하는 StorageClass가 target에도 있는지
- 없으면 `storageClassMappings`가 필요한지
- source volume이 `filesystem` 모드로 백업 가능한지
- `volumeBackupCompatibility.recommendedVolumeBackupMode` 와 `recommendedAction`



## 9. source backup 먼저 단독 검증

처음에는 통합 실행 전에 source backup만 먼저 단독 검증하는 것을 권장합니다.



API

- `POST /grasshopper/velero/source/backups`



권장 요청 포인트

- `namespace: "demo"`
- `volumeBackupMode: "filesystem"`
- `includedResources`에 필요한 리소스만 명시



예시 리소스

- `deployments`
- `services`
- `configmaps`
- `secrets`
- `persistentvolumes`,
- `persistentvolumeclaims`
- `pods`



### 9-1. backup 상태 확인



API

- `POST /grasshopper/velero/source/backups/{name}/validate`



정상 기대값

- `phase: Completed`



또는 source cluster에서 직접 확인

```bash
kubectl -n velero get backup
kubectl -n velero get backup backup-demo-fsb -o yaml
```



## 10. target restore 먼저 단독 검증

backup이 정상 완료되면 restore를 따로 검증합니다.

중요

- restore 생성 전에 target cluster에서 backup sync가 끝났는지 먼저 확인합니다.
- backup sync 전이면 restore API가 바로 실패하도록 바뀌었으니, 이 단계를 건너뛰지 않는 것이 좋습니다.



API

- `POST /grasshopper/velero/target/restores`



권장 요청 포인트

- `backupName: "backup-demo-fsb"`
- `sourceNamespace: "demo"`
- `targetNamespace: "demo-restored"`
- `storageClassMappings`
- `existingResourcePolicy: "update"`
- `restorePVs: true`

### 10-0. target cluster에서 backup sync 먼저 확인

아래처럼 target kubeconfig 기준으로 backup 조회가 먼저 성공해야 합니다.

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



### 10-1. restore 상태 확인



API

- `POST /grasshopper/velero/target/restores/{name}/validate`



정상 기대값

- `phase: Completed`



또는 target cluster에서 직접 확인

```bash
kubectl -n velero get restore
kubectl -n velero get restore restore-demo-fsb -o yaml
```



## 11. namespace remap 확인

이번 테스트에서는 `demo`를 `demo-restored`로 복구하는 것이 목표입니다.



확인 명령

```bash
kubectl get ns demo-restored
kubectl -n demo-restored get all
kubectl -n demo-restored get pvc
```



정상 기대값

- `demo-restored` namespace 존재
- `app` deployment 존재
- service/configmap/secret 존재
- PVC 존재



## 12. StorageClass mapping 확인

restore 요청에서 아래처럼 설정했다면

```json
"storageClassMappings": {
  "old-sc": "new-sc"
}
```

target PVC가 `new-sc`로 생성되었는지 확인합니다.



확인 명령

```bash
kubectl -n demo get pvc
kubectl -n demo-restored get pvc -o wide
```



정상 기대값

- source PVC는 기존 storage class
- target PVC는 `new-sc`



## 13. FSB 데이터 복구 확인

가장 중요한 단계입니다. source에 써둔 파일이 target에도 살아있는지 확인합니다.



사용 스크립트

- [verify-restored-data.sh](shared/verify-restored-data.sh)



실행

```bash
bash examples/kubernetes-velero/shared/verify-restored-data.sh
```



직접 확인

```bash
kubectl -n demo-restored exec deploy/app -- sh -c 'cat /usr/share/nginx/html/data/check.txt'
```



정상 기대값

```text
hello-velero
```

이 값이 보이면 FSB 기준 PVC 데이터 복구까지 성공한 것입니다.



## 14. existing resource policy 확인

이 옵션은 target에 이미 같은 이름의 리소스가 있을 때 동작을 확인하는 테스트입니다.

이번 구현에서는 `existingResourcePolicy: "update"`를 지원합니다.



테스트 예시

1. target cluster에 `demo-restored` namespace를 미리 생성
2. 같은 이름의 configmap 또는 deployment를 먼저 생성
3. restore 요청에 `existingResourcePolicy: "update"` 포함
4. restore 후 spec 반영 여부 확인



주의

- 이 기능은 리소스 종류에 따라 best effort입니다.
- PVC 데이터 overwrite를 일반 리소스 update처럼 단순하게 기대하면 안 됩니다.



## 15. 마지막으로 통합 실행 테스트

개별 단계가 모두 잘 되면 마지막에 통합 API를 실행합니다.



API

- `POST /grasshopper/velero/migration/execute`



이 API는 내부적으로 아래 순서로 동작합니다.

1. precheck
2. source backup 생성
3. backup 완료 대기
4. target restore 생성
5. restore 완료 대기



응답으로 `job_id`를 받으면 아래로 추적합니다.

```bash
curl http://localhost:8084/grasshopper/job/status/{{job_id}}
```

- `current_stage` 는 execute 흐름의 상위 단계 요약입니다.

```bash
curl http://localhost:8084/grasshopper/job/log/{{job_id}}
```



정상 기대값

- status: `completed`
- log에 backup 생성, restore 생성, 완료 메시지 순서대로 표시



## 16. 실패 시 확인 순서

문제가 생기면 아래 순서대로 확인하는 것이 좋습니다.

1. `job log`
   - `/grasshopper/job/log/{jobId}`
2. `backup validate`
3. `restore validate`
4. source/target cluster의 Velero pod 상태
5. source/target PVC 상태
6. source/target pod 이벤트



예시

```bash
kubectl -n velero get pods
kubectl -n velero logs deploy/velero
kubectl -n demo get pvc
kubectl -n demo-restored get pvc
kubectl -n demo-restored describe pod -l app=app
```



## 17. 권장 테스트 순서 요약

처음 테스트한다면 아래 순서가 가장 안전합니다.

1. source demo app 배포
2. source PVC에 테스트 데이터 기록
3. source install
4. target install
5. source health
6. target health
7. migration precheck
8. source backup create
9. backup validate
10. target restore create
11. restore validate
12. namespace remap 확인
13. storage class mapping 확인
14. restored PVC 데이터 확인
15. 마지막에 migration execute 통합 테스트

이 순서로 하면 어느 단계에서 문제가 생겼는지 분리해서 확인하기 쉽습니다.



## 18. kind 기반 로컬 2클러스터 테스트 방법

로컬에서 빠르게 검증하고 싶다면 `kind`로 source cluster와 target cluster를 각각 하나씩 띄워서 테스트할 수 있습니다.

전제

- Docker 실행 중
- `kind`, `kubectl` 설치 완료
- MinIO는 별도로 준비되어 있거나 로컬에서 접근 가능해야 함



### 18-1. kind cluster 생성

```bash
kind create cluster --name source-cluster
kind create cluster --name target-cluster
```

확인:

```bash
kubectl config get-contexts | grep kind
```

정상 기대값

- `kind-source-cluster`
- `kind-target-cluster`



### 18-2. kubeconfig 추출

source kubeconfig

```bash
kind get kubeconfig --name source-cluster > /tmp/source-kubeconfig.yaml
```

target kubeconfig

```bash
kind get kubeconfig --name target-cluster > /tmp/target-kubeconfig.yaml
```



base64 인코딩

```bash
base64 -i /tmp/source-kubeconfig.yaml | tr -d '\n'
```

```bash
base64 -i /tmp/target-kubeconfig.yaml | tr -d '\n'
```

이 값을 `api/http-client.env.json` 의 `local` 환경에 넣습니다.



### 18-3. 기본 StorageClass 확인

kind는 보통 기본 StorageClass가 하나 존재합니다. 먼저 source와 target 각각 확인합니다.

```bash
kubectl --kubeconfig /tmp/source-kubeconfig get sc
kubectl --kubeconfig /tmp/target-kubeconfig get sc
```

기본적으로는 `standard` 또는 환경에 따라 비슷한 이름이 보일 수 있습니다.

테스트를 단순하게 하려면

- source `storageClassName`도 `standard`
- target mapping도 `standard -> standard`

처럼 동일하게 두고 먼저 성공 경로를 확인하는 걸 권장합니다.



### 18-4. source demo app YAML 수정

[source-demo-app.yaml](shared/source-demo-app.yaml) 에서

```yaml
storageClassName: old-sc
```

를 source cluster의 실제 storage class로 바꿉니다.

예시

```yaml
storageClassName: standard
```



### 18-5. source cluster에 sample app 배포

```bash
kubectl --kubeconfig /tmp/source-kubeconfig apply -f examples/kubernetes-velero/shared/source-demo-app.yaml
```

확인

```bash
kubectl --kubeconfig /tmp/source-kubeconfig -n demo get all
kubectl --kubeconfig /tmp/source-kubeconfig -n demo get pvc
```



### 18-6. source PVC에 테스트 데이터 쓰기

스크립트는 현재 현재 kubectl context 기준으로 동작하므로 source context를 먼저 맞추고 실행합니다.

```bash
kubectl config use-context kind-source-cluster
bash examples/kubernetes-velero/shared/write-test-data.sh
```

또는 직접

```bash
kubectl --kubeconfig /tmp/source-kubeconfig -n demo exec deploy/app -- sh -c 'echo hello-velero > /usr/share/nginx/html/data/check.txt'
kubectl --kubeconfig /tmp/source-kubeconfig -n demo exec deploy/app -- sh -c 'cat /usr/share/nginx/html/data/check.txt'
```



### 18-7. cm-grasshopper API 요청 값 채우기

`api/http-client.env.json` 의 `local` 환경에 아래 값을 채웁니다.

1. `base64_source_kubeconfig`
2. `base64_target_kubeconfig`
3. `minio_url`
4. `minio_accesskey`
5. `minio_secretkey`

그리고 restore 요청에서 storage class mapping은 우선 동일 값으로 두는 것이 안전합니다.

예시

```json
"storageClassMappings": {
  "standard": "standard"
}
```



### 18-8. API 실행 순서

아래 순서를 그대로 따라가면 됩니다.

1. Source install
2. Target install
3. Source health
4. Target health
5. Migration precheck
6. Source backup create with FSB
7. Backup validate
8. Target restore create with namespace remap and storage class mapping
9. Restore validate
10. End-to-end migration execute



### 18-9. target cluster에서 복구 결과 확인

restore 후 target cluster context로 확인합니다.

```bash
kubectl --kubeconfig /tmp/target-kubeconfig get ns demo-restored
kubectl --kubeconfig /tmp/target-kubeconfig -n demo-restored get all
kubectl --kubeconfig /tmp/target-kubeconfig -n demo-restored get pvc
```



### 18-10. FSB 데이터 확인

```bash
kubectl --kubeconfig /tmp/target-kubeconfig -n demo-restored exec deploy/app -- sh -c 'cat /usr/share/nginx/html/data/check.txt'
```

정상 기대값

```text
hello-velero
```



### 18-11. kind 테스트에서 자주 보는 문제

1. MinIO endpoint 접근 실패
- kind cluster 내부/외부 네트워크 경로 차이 때문일 수 있음
- `cm-grasshopper`와 kind node가 같은 endpoint를 볼 수 있는지 확인 필요

2. PVC가 Bound 되지 않음
- kind 기본 storage provisioner 상태 확인
- `kubectl get sc`, `kubectl get pvc -A`

3. restore는 됐는데 파일이 없음
- source pod에 실제로 파일이 기록됐는지 확인
- backup validate / restore validate 먼저 확인
- velero node-agent pod 상태 확인

4. restore create가 바로 실패함
- `backup is not available on target cluster yet` 메시지면 target backup sync 대기 후 재시도
- `validationErrors`가 보이면 restore validate 또는 restore get 응답에서 상세 원인 확인

예시

```bash
kubectl --kubeconfig /tmp/source-kubeconfig -n velero get pods
kubectl --kubeconfig /tmp/target-kubeconfig -n velero get pods
```



### 18-12. kind 테스트 종료

테스트가 끝나면 cluster를 정리합니다.

```bash
kind delete cluster --name source-cluster
kind delete cluster --name target-cluster
```
