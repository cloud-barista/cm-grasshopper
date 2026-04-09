# kind + NFS StorageClass 테스트 가이드

이 문서는 `kind` 환경에서 `NFS StorageClass`를 사용해 `cm-grasshopper + Velero`의 `filesystem` 백업/복구를 테스트하는 방법을 정리합니다.

목표는 아래입니다.

- source kind 클러스터에서 PVC 데이터가 포함된 workload 배포
- Velero backup 수행
- target kind 클러스터로 restore 수행
- PVC, Pod, 파일 데이터까지 복구 확인



## 1. 전제

아래 도구가 준비되어 있어야 합니다.

- Docker
- kind
- kubectl
- cm-grasshopper
- base64 인코딩 가능한 쉘 환경

권장 파일 위치 예시는 아래처럼 가정합니다.

- 프로젝트 루트: `/Users/taking/Documents/innogrid/projects/cm-grasshopper`



## 2. kind 클러스터 생성

source, target 두 개의 kind 클러스터를 만듭니다.

```bash
kind create cluster --name source-cluster
kind create cluster --name target-cluster
```

확인

```bash
kubectl cluster-info --kubeconfig /tmp/source-kubeconfig
kubectl cluster-info --kubeconfig /tmp/target-kubeconfig
```



## 3. kubeconfig 추출

source, target 각각 kubeconfig를 파일로 저장합니다.

```bash
kind get kubeconfig --name source-cluster > /tmp/source-kubeconfig
kind get kubeconfig --name target-cluster > /tmp/target-kubeconfig
```

base64 인코딩

```bash
base64 < /tmp/source-kubeconfig | tr -d '\n' > /tmp/source-kubeconfig.b64
base64 < /tmp/target-kubeconfig | tr -d '\n' > /tmp/target-kubeconfig.b64
```



## 4. NFS 서버 컨테이너 실행

테스트용 NFS 서버를 Docker로 실행합니다.

중요

- `kind` 노드 컨테이너가 NFS 서버에 직접 접근해야 하므로, NFS 서버 컨테이너는 **반드시 Docker `kind` 네트워크에 연결**하는 것을 권장합니다.
- 일반 `docker compose` 기본 네트워크나 별도 bridge 네트워크에 붙이면 `mount.nfs: Connection refused`가 날 수 있습니다.
- 이 문서에서는 기본 예시로 `obeoneorg/nfs-server`를 사용합니다. 이 이미지는 `erichough/nfs-server` 계열의 multi-arch 포크입니다.
- `itsthenetwork/nfs-server-alpine`는 일부 ARM/OrbStack 환경에서 `Starting Mountd in the background...` 이후 재시작 루프를 타며 불안정할 수 있습니다.

`/exports`는 **컨테이너 내부 경로**입니다. 로컬 호스트에 `/exports` 폴더가 반드시 미리 있어야 하는 것은 아닙니다.

다만 테스트 중 데이터를 컨테이너 재생성 후에도 유지하고 싶다면, 로컬 폴더를 `/exports`에 mount 하는 방식을 권장합니다.



예시 `docker-compose.yml`

```yaml
services:
  kind-nfs-server:
    image: obeoneorg/nfs-server:latest
    container_name: kind-nfs-server
    restart: unless-stopped
    privileged: true
    environment:
      NFS_EXPORT_0: /exports *(rw,fsid=0,async,no_subtree_check,no_auth_nlm,insecure,no_root_squash)
    volumes:
      - ./nfs-data:/exports
    networks:
      - kind

networks:
  kind:
    external: true
```



위 예시에서는

- 컨테이너 내부 export 경로: `/exports`
- 로컬 호스트 경로: `./nfs-data`

즉 로컬에 필요한 경로는 `/exports`가 아니라 `./nfs-data`입니다.



compose 실행

```bash
mkdir -p ./nfs-data
docker compose up -d
```



위 설정을 쓰기 전에 `kind` 네트워크가 있는지 확인합니다.

```bash
docker network ls | grep kind
```



단순 1회 테스트용이면 `docker run` 방식으로도 충분합니다.

```bash
docker run -d \
  --name kind-nfs-server \
  --restart unless-stopped \
  --privileged \
  --network kind \
  -e 'NFS_EXPORT_0=/exports *(rw,fsid=0,async,no_subtree_check,no_auth_nlm,insecure,no_root_squash)' \
  -v "$(pwd)/tmp/nfs-data:/exports" \
  obeoneorg/nfs-server:latest
```



확인

```bash
docker ps | grep kind-nfs-server
docker logs kind-nfs-server
```



정상 로그 예시는 대략 아래와 비슷합니다.

```text
Exporting File System...
/exports             <world>
```



반대로 아래처럼 반복되면 이미지/런타임 조합이 불안정한 것입니다.

```text
Starting Mountd in the background...
Startup of NFS failed, sleeping for 2s, then retrying...
```



## 5. kind 네트워크에서 NFS 서버 주소 확인

host에서 접근하는 주소와 kind 노드 컨테이너에서 접근하는 주소는 다를 수 있으니, **`kind` 네트워크 기준 IP**를 확인합니다.

```bash
docker inspect kind-nfs-server --format '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}'
```

예시 결과

```text
192.168.147.10
```

이 값을 아래 예시에서 `NFS_SERVER_IP`로 사용합니다.

만약 NFS provisioner Pod 이벤트에 아래 같은 메시지가 보이면

```text
mount.nfs: Connection refused
```

대부분은 NFS 서버 컨테이너가 `kind` 네트워크에 없거나, `NFS_SERVER_IP`가 `kind` 네트워크 기준 IP가 아닌 경우입니다.

또는 NFS 서버 컨테이너 자체가 정상 기동하지 않은 경우일 수 있으니 `docker logs kind-nfs-server`를 같이 확인합니다.



## 6. source/target 클러스터에 NFS provisioner 설치

아래 manifest는 예시입니다. 실제 테스트에서는 `NFS_SERVER_IP`를 위에서 확인한 값으로 바꿔야 합니다.

먼저 namespace 생성

```bash
kubectl --kubeconfig /tmp/source-kubeconfig create namespace nfs-provisioner
kubectl --kubeconfig /tmp/target-kubeconfig create namespace nfs-provisioner
```

다음 내용을 `/tmp/nfs-provisioner.yaml`로 저장합니다.

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: nfs-client-provisioner
  namespace: nfs-provisioner
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nfs-client-provisioner-runner
rules:
- apiGroups: [""]
  resources: ["nodes", "persistentvolumes", "persistentvolumeclaims", "events", "services", "endpoints"]
  verbs: ["get", "list", "watch", "create", "delete", "patch", "update"]
- apiGroups: ["storage.k8s.io"]
  resources: ["storageclasses"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["persistentvolumeclaims/status"]
  verbs: ["update"]
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["extensions"]
  resources: ["podsecuritypolicies"]
  verbs: ["use"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: run-nfs-client-provisioner
subjects:
- kind: ServiceAccount
  name: nfs-client-provisioner
  namespace: nfs-provisioner
roleRef:
  kind: ClusterRole
  name: nfs-client-provisioner-runner
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nfs-client-provisioner
  namespace: nfs-provisioner
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nfs-client-provisioner
  template:
    metadata:
      labels:
        app: nfs-client-provisioner
    spec:
      serviceAccountName: nfs-client-provisioner
      containers:
      - name: nfs-client-provisioner
        image: registry.k8s.io/sig-storage/nfs-subdir-external-provisioner:v4.0.2
        env:
        - name: PROVISIONER_NAME
          value: kind.local/nfs
        - name: NFS_SERVER
          value: NFS_SERVER_IP
        - name: NFS_PATH
          value: /exports
        volumeMounts:
        - name: nfs-client-root
          mountPath: /persistentvolumes
      volumes:
      - name: nfs-client-root
        nfs:
          server: NFS_SERVER_IP
          path: /exports
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: nfs-client
provisioner: kind.local/nfs
reclaimPolicy: Delete
volumeBindingMode: Immediate
allowVolumeExpansion: true
```

`NFS_SERVER_IP`를 실제 값으로 바꾼 뒤 source, target에 각각 적용합니다.

```bash
kubectl --kubeconfig /tmp/source-kubeconfig apply -f /tmp/nfs-provisioner.yaml
kubectl --kubeconfig /tmp/target-kubeconfig apply -f /tmp/nfs-provisioner.yaml
```



## 7. NFS StorageClass 확인

```bash
kubectl --kubeconfig /tmp/source-kubeconfig get sc
kubectl --kubeconfig /tmp/target-kubeconfig get sc
```

둘 다 `nfs-client`가 보여야 합니다.



## 8. source 테스트 앱 배포

기존 예제 파일을 사용합니다.

- [source-demo-app.yaml](../../shared/source-demo-app.yaml)

배포 전에 `storageClassName`을 `nfs-client`로 바꿉니다.

```bash
kubectl --kubeconfig /tmp/source-kubeconfig apply -f examples/kubernetes-velero/shared/source-demo-app.yaml
```

확인

```bash
kubectl --kubeconfig /tmp/source-kubeconfig -n demo get pvc
kubectl --kubeconfig /tmp/source-kubeconfig -n demo get pods
```

PVC는 `Bound`, Pod는 `Running`이어야 합니다.



## 9. source PVC에 테스트 데이터 쓰기

기존 스크립트를 활용합니다.

- [write-test-data.sh](../../shared/write-test-data.sh)

필요하면 스크립트 안의 kubeconfig/context 사용 부분을 source kind 기준으로 맞춘 뒤 실행합니다.

예시

```bash
kubectl --kubeconfig /tmp/source-kubeconfig -n demo exec deploy/app -- sh -c 'echo hello-kind-nfs > /usr/share/nginx/html/data/check.txt'
```

확인

```bash
kubectl --kubeconfig /tmp/source-kubeconfig -n demo exec deploy/app -- cat /usr/share/nginx/html/data/check.txt
```



## 10. cm-grasshopper 실행

repo 루트에서 실행

```bash
make build-only
make run
```



## 11. Velero source/target install

예제 파일

- [01-install.http](../../api/01-install.http)
- [02-health.http](../../api/02-health.http)
- [03-precheck.http](../../api/03-precheck.http)
- [04-backup.http](../../api/04-backup.http)
- [05-restore.http](../../api/05-restore.http)

채워야 할 값

- `baseUrl`
- `base64_source_kubeconfig`
- `base64_target_kubeconfig`
- `minio_url`
- `minio_accesskey`
- `minio_secretkey`
- `minio_bucket`

먼저 아래 순서로 호출합니다.

1. `POST /velero/source/install`
2. `POST /velero/target/install`

각각 job id가 오면 상태 확인

```bash
curl http://localhost:8084/grasshopper/job/status/{{job_id}}
curl http://localhost:8084/grasshopper/job/log/{{job_id}}
```



## 12. migration precheck

`precheck.volumeBackupMode`는 `filesystem`으로 둡니다.

기대 포인트

- `status: ready` 또는 `ready_with_warnings`
- `source.volumeBackupCompatibility.filesystemBackupReady: true`
- `recommendedVolumeBackupMode: "filesystem"`

만약 여기서 `not_ready`가 나오면 backup/restore를 진행하지 않는 게 좋습니다.



## 13. source backup 생성

`POST /velero/source/backups`

권장 body 포인트

```json
{
  "backup": {
    "sourceNamespace": "demo",
    "volumeBackupMode": "filesystem",
    "nameConflictPolicy": "rename"
  }
}
```

생성 후 확인

```bash
velero backup describe <backup-name> --details --kubeconfig /tmp/source-kubeconfig
kubectl --kubeconfig /tmp/source-kubeconfig get podvolumebackups -n velero
```

정상 기대값

- `Pod Volume Backups`가 비어 있지 않음
- `kubectl get podvolumebackups -n velero` 결과가 존재함



## 14. target restore 생성

`POST /velero/target/restores`

권장 body 포인트

```json
{
  "restore": {
    "backupName": "<backup-name>",
    "sourceNamespace": "demo",
    "targetNamespace": "demo-restored",
    "storageClassMappings": {
      "nfs-client": "nfs-client"
    },
    "existingResourcePolicy": "update",
    "restorePVs": true
  }
}
```

확인

```bash
velero restore describe <restore-name> --details --kubeconfig /tmp/target-kubeconfig
kubectl --kubeconfig /tmp/target-kubeconfig get podvolumerestores -n velero
```

정상 기대값

- `PodVolumeRestores`가 존재함
- restore phase가 `Completed`



## 15. target 결과 확인

PVC와 Pod 확인

```bash
kubectl --kubeconfig /tmp/target-kubeconfig -n demo-restored get pvc
kubectl --kubeconfig /tmp/target-kubeconfig -n demo-restored get pods
```

기대값

- PVC `Bound`
- Pod `Running`

파일 데이터 확인

```bash
kubectl --kubeconfig /tmp/target-kubeconfig -n demo-restored exec deploy/app -- cat /usr/share/nginx/html/data/check.txt
```

`hello-kind-nfs`가 나오면 성공입니다.



## 16. 실패 시 가장 먼저 볼 것

1. source backup describe
2. source `podvolumebackups`
3. target restore describe
4. target `podvolumerestores`
5. source/target `kubectl -n velero get pods`
6. target PVC/PV 상태



## 17. 권장 판단

- `kind` 기본 `local-path`는 PVC 데이터 migration 검증용으로 비추천
- `NFS StorageClass`는 `filesystem` 테스트용으로 적합
- `CSI snapshot` 테스트가 목적이면 별도 CSI hostpath 구성이 더 적합
