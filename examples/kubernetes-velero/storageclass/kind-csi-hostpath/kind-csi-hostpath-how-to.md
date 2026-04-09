# kind + CSI hostpath 테스트 가이드

이 문서는 `kind` 환경에서 `CSI hostpath driver`를 사용해 `cm-grasshopper + Velero`의 `snapshot` 테스트를 진행하는 방법을 정리합니다.

목표는 아래입니다.

- source kind 클러스터에 CSI snapshot 가능한 StorageClass 구성
- target kind 클러스터에도 동일한 CSI snapshot 환경 구성
- `volumeBackupMode: "snapshot"` 기준으로 backup/restore 테스트
- PVC, Pod, 파일 데이터까지 복구 확인



## 중요한 제한

- `kind + csi-hostpath` 조합은 **단일 클러스터 내 CSI snapshot 생성/백업 동작 확인용**으로는 유용합니다.
- 하지만 **source kind -> target kind** 처럼 서로 다른 클러스터 사이의 snapshot restore는 성공을 보장하지 않습니다.
- 이유는 `hostpath.csi.k8s.io`가 source cluster에서 만든 snapshot handle을 target cluster의 독립적인 backend에서 그대로 찾을 수 없기 때문입니다.
- 즉 이 환경에서는
  - source cluster에서 CSI snapshot backup 동작 확인 : 적합
  - target cluster에서 cross-cluster snapshot restore 성공 검증 : 비권장
- cross-cluster 복구까지 실제로 검증하려면, source/target이 같은 실제 스토리지 backend를 공유하거나 snapshot portability를 제공하는 CSI 스토리지를 써야 합니다.



## 1. 전제

아래 도구가 준비되어 있어야 합니다.

- Docker
- kind
- kubectl
- cm-grasshopper
- git

이 문서는 예시 기준으로 Kubernetes CSI hostpath driver 배포 매니페스트를 사용합니다.



## 2. kind 클러스터 생성

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

```bash
kind get kubeconfig --name source-cluster > /tmp/source-kubeconfig
kind get kubeconfig --name target-cluster > /tmp/target-kubeconfig
base64 < /tmp/source-kubeconfig | tr -d '\n' > /tmp/source-kubeconfig.b64
base64 < /tmp/target-kubeconfig | tr -d '\n' > /tmp/target-kubeconfig.b64
```



## 4. external-snapshotter 저장소 준비

snapshot controller도 개별 raw YAML 적용보다 공식 저장소 기준으로 설치하는 것을 권장합니다.

```bash
cd /tmp
git clone https://github.com/kubernetes-csi/external-snapshotter.git
cd external-snapshotter
git checkout tags/v8.5.0
```



## 5. source cluster에 snapshot controller 설치

```bash
cd /tmp/external-snapshotter
kubectl --kubeconfig /tmp/source-kubeconfig kustomize client/config/crd | kubectl --kubeconfig /tmp/source-kubeconfig apply -f -
kubectl --kubeconfig /tmp/source-kubeconfig -n kube-system kustomize deploy/kubernetes/snapshot-controller | kubectl --kubeconfig /tmp/source-kubeconfig apply -f -
```



## 6. target cluster에 snapshot controller 설치

```bash
cd /tmp/external-snapshotter
kubectl --kubeconfig /tmp/target-kubeconfig kustomize client/config/crd | kubectl --kubeconfig /tmp/target-kubeconfig apply -f -
kubectl --kubeconfig /tmp/target-kubeconfig -n kube-system kustomize deploy/kubernetes/snapshot-controller | kubectl --kubeconfig /tmp/target-kubeconfig apply -f -
```



## 7. csi-driver-host-path 저장소 준비

공식 hostpath driver 저장소를 사용합니다.

```bash
cd /tmp
git clone https://github.com/kubernetes-csi/csi-driver-host-path.git
cd csi-driver-host-path
git checkout tags/v1.17.1
```



## 8. source cluster에 CSI hostpath driver 설치

```bash
cd /tmp/csi-driver-host-path
KUBECONFIG=/tmp/source-kubeconfig ./deploy/kubernetes-latest/deploy.sh
kubectl --kubeconfig /tmp/source-kubeconfig apply -f https://github.com/kubernetes-csi/csi-driver-host-path/raw/refs/tags/v1.17.1/examples/csi-storageclass.yaml
```



## 9. target cluster에 CSI hostpath driver 설치

```bash
cd /tmp/csi-driver-host-path
KUBECONFIG=/tmp/target-kubeconfig ./deploy/kubernetes-latest/deploy.sh
kubectl --kubeconfig /tmp/target-kubeconfig apply -f https://github.com/kubernetes-csi/csi-driver-host-path/raw/refs/tags/v1.17.1/examples/csi-storageclass.yaml
```

참고

- 이 스크립트는 CSI hostpath driver, StorageClass, VolumeSnapshotClass, 관련 컴포넌트를 함께 배포하는 데 더 적합합니다.
- Pod만 뜨고 `CSIDriver`가 생성되지 않는 문제를 줄이는 데 유리합니다.



## 10. 설치 확인

```bash
# source
kubectl --kubeconfig /tmp/source-kubeconfig get pods -A | grep -i hostpath
kubectl --kubeconfig /tmp/source-kubeconfig get csidriver
kubectl --kubeconfig /tmp/source-kubeconfig get sc
kubectl --kubeconfig /tmp/source-kubeconfig get volumesnapshotclass
# target
kubectl --kubeconfig /tmp/target-kubeconfig get pods -A | grep -i hostpath
kubectl --kubeconfig /tmp/target-kubeconfig get csidriver
kubectl --kubeconfig /tmp/target-kubeconfig get sc
kubectl --kubeconfig /tmp/target-kubeconfig get volumesnapshotclass
```

정상 기대값

- hostpath 관련 Pod가 `Running`
- `kubectl get csidriver` 결과에 `hostpath.csi.k8s.io` 존재
- `csi-hostpath-sc` 존재
- `VolumeSnapshotClass` 존재

`CSIDriver`가 비어 있으면 설치가 불완전한 것이고, 그 상태에서는 PVC가 `Pending`으로 남을 수 있습니다.



## 10-1. VolumeSnapshotClass 라벨 적용

Velero가 사용할 `VolumeSnapshotClass`를 명확하게 지정하려면 source/target 모두 아래 라벨을 붙이는 것을 권장합니다.

```bash
kubectl --kubeconfig /tmp/source-kubeconfig label volumesnapshotclass csi-hostpath-snapclass velero.io/csi-volumesnapshot-class=true --overwrite
kubectl --kubeconfig /tmp/target-kubeconfig label volumesnapshotclass csi-hostpath-snapclass velero.io/csi-volumesnapshot-class=true --overwrite
```

확인

```bash
kubectl --kubeconfig /tmp/source-kubeconfig get volumesnapshotclass csi-hostpath-snapclass -o yaml
kubectl --kubeconfig /tmp/target-kubeconfig get volumesnapshotclass csi-hostpath-snapclass -o yaml
```

기대값

- `metadata.labels.velero.io/csi-volumesnapshot-class: "true"`



## 11. source 테스트 앱 배포

기존 예제 파일을 사용합니다.

- [source-demo-app.yaml](../../shared/source-demo-app.yaml)

배포 전에 `storageClassName`을 CSI hostpath용 StorageClass 이름으로 바꿉니다.

```bash
kubectl --kubeconfig /tmp/source-kubeconfig apply -f examples/kubernetes-velero/shared/source-demo-app.yaml
```

확인

```bash
kubectl --kubeconfig /tmp/source-kubeconfig -n demo get pvc
kubectl --kubeconfig /tmp/source-kubeconfig -n demo get pods
```



## 12. source PVC 상태 먼저 확인

```bash
kubectl --kubeconfig /tmp/source-kubeconfig -n demo get pvc
kubectl --kubeconfig /tmp/source-kubeconfig -n demo describe pvc demo-data
```

정상 기대값

- PVC `Bound`
- PV 이름이 표시됨

아래처럼 계속 `Pending`이면 Velero 테스트로 넘어가지 말고 CSI 설치부터 다시 점검해야 합니다.

- `Waiting for a volume to be created by the external provisioner 'hostpath.csi.k8s.io'`



## 13. source PVC에 테스트 데이터 쓰기

```bash
kubectl --kubeconfig /tmp/source-kubeconfig -n demo exec deploy/app -- sh -c 'echo hello-kind-csi > /usr/share/nginx/html/data/check.txt'
kubectl --kubeconfig /tmp/source-kubeconfig -n demo exec deploy/app -- cat /usr/share/nginx/html/data/check.txt
```



## 14. cm-grasshopper 실행

```bash
make build-only
make run
```



## 15. Velero source/target install

예제 파일

- [01-install.http](../../api/01-install.http)
- [02-health.http](../../api/02-health.http)
- [03-precheck.http](../../api/03-precheck.http)
- [04-backup.http](../../api/04-backup.http)
- [05-restore.http](../../api/05-restore.http)

주의

현재 `cm-grasshopper` 기본 Velero 설치는 FSB 중심입니다. `snapshot` 테스트를 하려면 Velero install 옵션이나 배포 설정에서 CSI snapshot 구성이 같이 맞아야 합니다.

즉 이 문서는 **kind에서 snapshot 가능한 CSI 환경을 구성하는 방법** 중심이고, 실제 snapshot 테스트는 Velero install 설정이 CSI 지원으로 정렬돼 있어야 합니다.



## 16. precheck

`precheck`는 아래처럼 `volumeBackupMode: "snapshot"`으로 요청합니다.

기대 포인트

- source/target cluster 접근 가능
- StorageClass mapping 확인 가능
- source PVC 존재



## 17. source backup

권장 body

```json
{
  "backup": {
    "sourceNamespace": "demo",
    "volumeBackupMode": "snapshot",
    "nameConflictPolicy": "rename"
  }
}
```

확인

```bash
velero backup describe <backup-name> --details --kubeconfig /tmp/source-kubeconfig
kubectl --kubeconfig /tmp/source-kubeconfig get volumesnapshot -A
kubectl --kubeconfig /tmp/source-kubeconfig get volumesnapshotcontent
```

정상 기대값

- backup 완료
- source 쪽 snapshot backup 동작 확인

참고

- backup finalizer 단계에서 `VolumeSnapshot`과 `VolumeSnapshotContent`가 삭제될 수 있으므로, `kubectl get volumesnapshot -A`가 비어 있어도 로그에서 생성/삭제 흔적을 확인하는 것이 더 정확합니다.



## 18. target restore

restore 예시는 일반 restore와 동일하지만, snapshot 기반이면 source snapshot 정보와 target CSI 드라이버 호환성이 중요합니다.

중요

- `kind + csi-hostpath`에서는 target cluster가 source snapshot handle을 찾지 못해 `snapshot ... is not Ready` 오류가 날 수 있습니다.
- 이건 문서상의 설치 누락이 아니라, `kind` 두 클러스터가 서로 다른 독립 backend를 가지는 구조적 한계에 가깝습니다.

확인

```bash
velero restore describe <restore-name> --details --kubeconfig /tmp/target-kubeconfig
kubectl --kubeconfig /tmp/target-kubeconfig -n demo-restored get pvc
kubectl --kubeconfig /tmp/target-kubeconfig get pv
```



## 19. 데이터 확인

```bash
kubectl --kubeconfig /tmp/target-kubeconfig -n demo-restored exec deploy/app -- cat /usr/share/nginx/html/data/check.txt
```

`hello-kind-csi`가 보이면 성공입니다.



## 20. 권장 판단

- `kind + CSI hostpath`는 단일 클러스터 snapshot backup 동작 확인용으로는 좋음
- `kind + CSI hostpath`는 cross-cluster snapshot restore 검증용으로는 비권장
- `filesystem`만 검증하려면 `kind + NFS`가 더 단순함
- 현재 `cm-grasshopper` 기본 설치는 FSB 중심이므로, snapshot 중심 테스트는 Velero install 방향을 따로 맞추는 게 좋음
