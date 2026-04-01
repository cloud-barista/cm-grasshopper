# Kubernetes Velero Examples

## Folders

- root `.md`
  - 전체 개요, API 설명, step-by-step 문서
- `api/`
  - VS Code REST Client용 `rest-client.env.json`, flow 순서 `.http`
- `shared/`
  - 여러 가이드에서 공통으로 쓰는 demo app, 데이터 쓰기/검증 스크립트, storage class mapping 예시
- `storageclass/kind-nfs/`
  - `kind + NFS StorageClass` 기반 filesystem 백업/복구 테스트 가이드
- `storageclass/kind-csi-hostpath/`
  - `kind + CSI hostpath` 기반 snapshot 테스트 가이드



## Quick start

### 1. source cluster에 sample app 배포

`shared/source-demo-app.yaml`의 `storageClassName`을 source 환경에 맞게 바꾼 뒤:

```bash
kubectl apply -f examples/kubernetes-velero/shared/source-demo-app.yaml
```



### 2. source PVC에 테스트 데이터 기록

```bash
bash examples/kubernetes-velero/shared/write-test-data.sh
```



### 3. Velero install / precheck / backup / restore / execute

`velero-migration-api.md`, `velero-migration-how-to.md`, `api/` 폴더의 flow 순서 `.http` 파일 기준으로 실행

- 공통 환경값: `api/rest-client.env.json` 의 `local` 환경

- install : `api/01-install.http`
- health : `api/02-health.http`
- precheck : `api/03-precheck.http`
- source backup : `api/04-backup.http`
- target restore : `api/05-restore.http`
- migration execute : `api/06-execute.http`
- job 조회 : `api/07-job.http`



### 4. target cluster에서 복구 결과 확인

```bash
bash examples/kubernetes-velero/shared/verify-restored-data.sh
```



## Notes

- `old-sc`는 source cluster의 실제 storage class로 수정 필요
- `new-sc`는 target cluster의 실제 storage class로 매핑 필요
- `targetNamespace=demo-restored` 기준으로 검증 스크립트가 작성돼 있음
- FSB 기준으로는 Pod가 PVC를 마운트하고 있어야 데이터 검증이 쉽습니다
