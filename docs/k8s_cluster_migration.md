# 쿠버네티스 클러스터 마이그레이션 — kubeconfig 해석과 CSP 인증

이 문서는 Cloud-Barista 마이그레이션 체계에서 **cm-grasshopper** 가 수행하는 **쿠버네티스(k8s) 클러스터 마이그레이션**의 동작 방식과, 그 과정에서 가장 까다로운 **CSP 별 kubeconfig 인증 차이**를 어떻게 해결했는지 정리한 자료입니다.

> 핵심 요약: cm-grasshopper 는 velero 로 source→S3→target 백업/복원을 수행합니다. 이때 두 클러스터에 접근할 **kubeconfig** 가 필요한데, cb-tumblebug 가 주는 kubeconfig 는 CSP 마다 인증 방식이 달라서 그대로는 컨테이너 안에서 동작하지 않습니다. cm-grasshopper 는 cb-tumblebug 의 k8s 정보를 받아 **exec-plugin 형 kubeconfig 를 "토큰 브로커(broker-exec)" 형태로 재작성**하여 클라우드 CLI·자격증명 없이 동작하게 만듭니다.

---

## 0. 한눈에 보기

| 항목 | 내용 |
|---|---|
| 마이그레이션 엔진 | **velero** (백업/복원), Helm 으로 설치 |
| 데이터 경유지 | **S3 호환 오브젝트 스토리지** (RustFS/MinIO/AWS S3 …) |
| 클러스터 접근 | **kubeconfig** (요청에 직접 전달 **또는** cb-tumblebug 참조) |
| 핵심 문제 | CSP 마다 kubeconfig 인증 방식이 다름 (임베드 vs exec-plugin) |
| 해결 | exec-plugin 형은 cb-tumblebug `/token` 을 호출하는 **broker-exec** 로 재작성 |
| 수정 범위 | **cm-grasshopper 만** 수정 (cm-honeybee/cb-tumblebug/cb-spider 변경 없음) |

---

## 1. 사전 준비 — cb-tumblebug

cm-grasshopper 의 k8s 마이그레이션은 cb-tumblebug 가 **이미 k8s 클러스터를 배포·관리하고 있는 상태**를 전제로 합니다. 아래는 그 상태를 만드는 절차입니다. (배포 세부는 cb-tumblebug 공식 문서/`README.md`·`init/README.md` 참고)

### 1.1 cb-tumblebug 기동

docker-compose 로 cb-tumblebug + cb-spider + etcd + postgres + (mc-terrarium/openbao) 스택을 띄웁니다. REST API 는 기본 `:1323`, **Basic Auth** 를 씁니다.

```bash
# cb-tumblebug 레포에서
make up        # 서비스 기동 (최초 1회 OpenBao 자동 init)
```

- API: `http://<host>:1323/tumblebug/...`
- 인증: `.env` 의 `TB_API_USERNAME` / `TB_API_PASSWORD` (예: `default` / `default`)
- 준비 확인: `curl -u default:default http://<host>:1323/tumblebug/readyz`

### 1.2 CSP 자격증명 등록 (init)

```bash
make gen-cred                              # 템플릿 → ~/.cloud-barista/credentials.yaml
vi  ~/.cloud-barista/credentials.yaml      # AWS/Azure 등 자격증명 입력
make enc-cred                              # 암호화 (credentials.yaml.enc)
make init                                  # 자격증명 등록 + 스펙/이미지 로드 + 템플릿 등록
```

`credentials.yaml` 구조(요약):

```yaml
credentialholder:
  admin:
    aws:   { aws_access_key_id: <KEY>, aws_secret_access_key: <SECRET> }
    azure: { clientId: <...>, clientSecret: <...>, tenantId: <...>, subscriptionId: <...> }
```

> 자격증명은 hybrid(RSA+AES) 암호화로 `POST /tumblebug/credential` 에 등록됩니다. `make init` 이 이 과정을 대신 처리합니다. holder 이름은 소문자/숫자/언더스코어만 허용(하이픈 불가).

등록·연결 확인:

```bash
curl -u default:default "http://<host>:1323/tumblebug/connConfig" \
  | jq -r '.connectionconfig[] | select(.providerName=="aws") | "\(.configName)\t\(.verified)"'
# aws-ap-northeast-2   true   ← verified=true 면 사용 가능
```

### 1.3 네임스페이스 생성

```bash
curl -u default:default -X POST "http://<host>:1323/tumblebug/ns" \
  -H 'Content-Type: application/json' \
  -d '{"name":"testns01","description":"k8s migration test"}'
```

### 1.4 k8s 클러스터 배포 (dynamic)

CSP 별 제약을 먼저 확인합니다.

```bash
H="http://<host>:1323"; U="-u default:default"
curl $U "$H/tumblebug/availableK8sVersion?providerName=aws&regionName=ap-northeast-2"
curl $U "$H/tumblebug/requiredK8sSubnetCount?providerName=aws"          # AWS: 2
curl $U "$H/tumblebug/checkK8sNodeGroupsOnK8sCreation?providerName=aws" # AWS: false → 생성 후 노드그룹 추가
```

동적 생성(공유 VNet/서브넷을 자동 구성). 아래는 실제로 EKS 를 띄울 때 사용한 요청입니다.

```bash
curl $U -X POST "$H/tumblebug/ns/testns01/k8sClusterDynamic" \
  -H 'Content-Type: application/json' -d '{
    "name":"ekstest01","version":"1.33","nodeGroupName":"ng1",
    "specId":"aws+ap-northeast-2+t3.medium","imageId":"default",
    "rootDiskType":"default","rootDiskSize":0,
    "onAutoScaling":"true","desiredNodeSize":1,"minNodeSize":1,"maxNodeSize":2,
    "connectionName":"aws-ap-northeast-2"
  }'
```

- `rootDiskSize` 는 **정수**(문자열 아님)로 보내야 함. EKS 는 control plane 이 먼저 `Active` 가 되고 노드그룹이 뒤따라 붙습니다.
- 상태 확인: `curl $U "$H/tumblebug/ns/testns01/k8sCluster/ekstest01"` → `status: Active`
- Azure 예: `specId: azure+koreacentral+standard_b2s_v2`, `connectionName: azure-koreacentral`.

### 1.5 접근 정보 확인 (cm-grasshopper 가 사용할 값)

```bash
curl $U "$H/tumblebug/ns/testns01/k8sCluster/ekstest01/kubeconfig" | jq -r '.kubeconfig' | head
curl $U "$H/tumblebug/ns/testns01/k8sCluster/ekstest01/token"      | jq .
```

- Azure AKS → kubeconfig 가 자체완결(인증서/토큰 임베드)
- AWS EKS / GCP GKE / NCP NKS → kubeconfig 가 exec-plugin (다음 장에서 cm-grasshopper 가 broker-exec 로 변환)

이 시점부터는 `(namespaceId, k8sClusterId)` 만 있으면 cm-grasshopper 가 나머지(kubeconfig 해석·인증)를 처리합니다. 마지막으로 cm-grasshopper 설정의 `tumblebug` 항목이 이 cb-tumblebug 를 가리키게 하면 됩니다(10장 10.1 참고).

---

## 2. 전체 흐름

```
[cm-grasshopper]
   │ 1) (선택) cb-tumblebug 에서 kubeconfig 해석
   │    GET /tumblebug/ns/{ns}/k8sCluster/{id}/kubeconfig
   ▼
[source k8s] ──velero backup──▶ [S3 버킷] ──velero restore──▶ [target k8s]
                                   ▲
                                   │ AccessKey/SecretKey/Endpoint
                              (S3Access)
```

velero 마이그레이션 API (요약):

| 메서드 | 경로 | 설명 |
|---|---|---|
| POST | `/grasshopper/velero/{role}/install` | source/target 에 velero 설치 |
| POST | `/grasshopper/velero/migration/precheck` | 사전 점검 |
| POST | `/grasshopper/velero/migration/execute` | 백업+복원 실행(비동기 Job) |

요청 본문은 `MultiClusterEnvelope` 로, `sourceCluster`/`targetCluster`(각각 `ClusterAccess`)와 `storage`(`S3Access`)를 담습니다.

---

## 3. cb-tumblebug 에서 가져오는 정보

cb-tumblebug 가 배포·관리하는 k8s 클러스터(관리형 PMK: AKS/EKS/GKE/NKS 등)는 아래 엔드포인트로 접근 정보를 제공합니다 (basic auth).

| 메서드 | 경로 | 반환 |
|---|---|---|
| GET | `/tumblebug/ns/{ns}/k8sCluster/{id}` | 클러스터 메타(`providerName`, `version`, `network`, `nodeGroup`, `status`, `accessInfo` …) |
| GET | `/tumblebug/ns/{ns}/k8sCluster/{id}/kubeconfig` | **kubeconfig** (YAML 문자열) |
| GET | `/tumblebug/ns/{ns}/k8sCluster/{id}/token` | **ExecCredential 베어러 토큰** |

cm-grasshopper 는 이 중 `/kubeconfig` 와 `/token` 을 사용합니다. 인증/주소는 기존 설정값 `cm-grasshopper.tumblebug.{server_address,server_port,username,password}` 를 그대로 재사용합니다.

---

## 4. 핵심 문제 — CSP 별 kubeconfig 인증 차이

cb-tumblebug(내부적으로 cb-spider)가 돌려주는 kubeconfig 는 CSP 에 따라 인증 방식이 다릅니다.

### 4.1 Azure AKS — 자체 완결형 (그대로 사용 가능 ✅)

```yaml
users:
- name: clusterAdmin_...
  user:
    client-certificate-data: <...>   # 인증서/키/토큰이 kubeconfig 안에 임베드
    client-key-data: <...>
    token: <...>
```

별도 CLI·자격증명이 필요 없으므로 cm-grasshopper 가 **그대로 사용**합니다.

### 4.2 AWS EKS / GCP GKE / NCP NKS — exec-plugin 형 (그대로는 실패 ❌)

실제 EKS 에서 확인한 kubeconfig:

```yaml
users:
- name: aws-iam-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      interactiveMode: Never
      command: aws-iam-authenticator   # ← 외부 바이너리 필요
      args: [token, -i, <clusterName>]
```

- kubeconfig 안에 토큰이 없고, kubectl/client-go 실행 시점에 **외부 바이너리**(`aws-iam-authenticator`, `gke-gcloud-auth-plugin` 등)를 실행해 토큰을 받습니다.
- 그 바이너리는 다시 **클라우드 자격증명**(AWS 키, GCP ADC 등)을 요구합니다.
- cm-grasshopper 컨테이너에는 이런 CLI 도, 클라우드 자격증명도 없으므로 **그대로는 100% 실패**합니다.

### 4.3 정리

| CSP | kubeconfig 인증 | 외부 CLI 필요 | CLI 없는 컨테이너에서 그대로 동작 | `/token` 지원 |
|---|---|---|---|---|
| **Azure AKS** | 인증서/토큰 임베드 | ❌ | ✅ | 없음 |
| **AWS EKS** | exec-plugin | ✅ | ❌ | ✅ |
| **GCP GKE** | exec-plugin | ✅ | ❌ | ✅ |
| **NCP NKS** | exec-plugin | ✅ | ❌ | ✅ |
| NHN | CSP-native passthrough | 추정 ✅ | ⚠️ | 없음 |

---

## 5. 해결 방식 — broker-exec (토큰 브로커)

### 5.1 아이디어

client-go 는 kubeconfig 의 `user.exec` 를 **네이티브로 지원**하여, API 호출 시점에 그 command 를 실행해 토큰을 받고 **만료/401 시 자동 재실행**해 갱신합니다. 이 "자동 갱신" 덕분에 velero 백업/복원처럼 오래 걸리는 작업에 유리합니다.

그래서 exec 자체를 없애지 않고, **exec command 만 클라우드 CLI 대신 cb-tumblebug `/token` 호출로 바꿉니다.** 그러면:

- 컨테이너에 `sh` + `curl` + `jq` 만 있으면 됨 (클라우드 CLI·자격증명 불필요)
- 모든 exec-plugin CSP 를 **동일한 로직**으로 처리
- client-go 의 자동 갱신을 그대로 활용

### 5.2 재작성 결과 kubeconfig

```yaml
users:
- name: <원래 사용자명>
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      interactiveMode: Never
      command: sh
      args:
      - -c
      - curl -fsS -u '<tbUser>:<tbPass>' 'http://<tb>/tumblebug/ns/<ns>/k8sCluster/<id>/token' | jq -ce .execCredential
```

> **왜 `jq` 가 필요한가?** cb-tumblebug 의 `/token` 응답은 `{"execCredential": {...}}` 로 한 번 감싸져 있는데, client-go 는 **bare ExecCredential** 객체를 stdout 으로 기대합니다. `jq -ce .execCredential` 로 껍데기를 벗겨 줍니다.

### 5.3 동작 판단 로직

```
cb-tumblebug /kubeconfig 조회
        │
        ▼
  user.exec 스탠자가 있는가?
   ├── 예(EKS/GKE/NCP) → 모든 exec authInfo 를 broker-exec 로 재작성
   └── 아니오(Azure)   → 원본 kubeconfig 그대로 사용
```

### 5.4 런타임 동작 흐름 (실제 호출 체인)

요청 1건을 처리할 때 코드가 실제로 거치는 경로입니다. velero 구현자는 이 흐름을 기준으로 동작/장애를 추적하면 됩니다.

```
velero 컨트롤러 (precheck / install / execute)
   │
   ▼
k8sclient.NewRESTConfig(cluster)                         ← lib/k8s/client/factory.go
   │
   ▼
k8scommon.ResolveKubeconfig(cluster)                     ← lib/k8s/common/validation.go
   ├─ cluster.Kubeconfig 있음        → DecodeKubeconfig (그대로/ base64 디코드)
   └─ cluster.Tumblebug 참조         → tumblebug.ResolveKubeconfig(ns, id)   ← lib/k8s/tumblebug/resolve.go
        1) GET http://<TB>/tumblebug/ns/<ns>/k8sCluster/<id>/kubeconfig  (basic-auth)
        2) clientcmd.Load 로 파싱
        3) user.exec 있으면 → broker-exec 로 재작성 / 없으면 원본 반환
   │
   ▼
clientcmd.RESTConfigFromKubeConfig(해석된 kubeconfig) → *rest.Config
   │
   ▼
clientset / controller-runtime client / Helm action config 생성
   │
   ▼ (EKS/GKE/NCP 인 경우, API 호출 시점마다)
client-go 가 kubeconfig 의 exec 명령(sh -c "curl … /token | jq …")을 실행
   → cb-tumblebug /token 에서 베어러 토큰 수령 → Authorization 헤더로 사용
   → 토큰 만료/401 시 client-go 가 명령을 자동 재실행해 갱신
```

> **요점**: `NewRESTConfig` 가 단일 관문이라, velero 의 health/install/precheck/execute 어느 경로든 동일하게 해석을 거칩니다. velero 구현 코드는 여전히 `commonmodel.ClusterAccess` 만 다루면 되고, kubeconfig 해석은 신경 쓸 필요가 없습니다.

---

## 6. 구현 내용 (cm-grasshopper 변경 파일)

| 파일 | 변경 |
|---|---|
| `pkg/api/rest/model/common/k8s.go` | `ClusterAccess` 에 `Tumblebug *TumblebugK8sRef{NamespaceID, K8sClusterID}` 추가 |
| `lib/k8s/tumblebug/resolve.go` *(신규)* | `ResolveKubeconfig(nsID, k8sClusterID)` — TB `/kubeconfig` 조회 → exec 면 broker-exec 재작성, 아니면 그대로 |
| `lib/k8s/common/validation.go` | `ResolveKubeconfig(cluster)`(raw vs TB 분기) 추가, `ValidateClusterAccess` 가 TB 참조도 허용 |
| `lib/k8s/client/factory.go` | `NewRESTConfig` 이 `ResolveKubeconfig` 사용 → clientset/controller/helm 전부 자동 적용 |
| `lib/k8s/installer/velero.go` | Helm 용 `ClusterAccess` 재조립 시 `Tumblebug` 참조 전달 |
| `Dockerfile` | prod 스테이지에 `jq` 추가 |
| `lib/k8s/tumblebug/resolve_test.go` *(신규)* | EKS(재작성)/AKS(그대로) 단위 테스트 |

해석은 `factory.NewRESTConfig` 라는 단일 지점에 들어가므로, k8s clientset·controller-runtime client·Helm action config 가 모두 동일하게 혜택을 받습니다.

> **cm-honeybee 는 왜 안 고쳤나?** honeybee 의 k8s 수집기는 컨트롤플레인 노드의 `/etc/kubernetes/admin.conf` 에 SSH 로 접근하는 **self-managed(kubeadm) 클러스터 전용**입니다. cb-tumblebug 가 배포한 **관리형 클러스터(AKS/EKS/GKE)는 컨트롤플레인이 CSP 에 숨겨져** 있어 honeybee 가 introspect 할 수 없고, 접근 정보는 cb-tumblebug 에서 받습니다. 따라서 이번 작업은 cm-grasshopper 안에서 자기완결되며 honeybee 수정이 필요 없습니다.

---

## 7. 사용법

### 7.1 cb-tumblebug 클러스터 참조 방식 (권장)

`kubeconfig` 대신 클러스터 참조만 넘기면 됩니다.

```json
{
  "sourceCluster": { "tumblebug": { "namespaceId": "testns01", "k8sClusterId": "ekstest01" } },
  "targetCluster": { "tumblebug": { "namespaceId": "testns01", "k8sClusterId": "k8stest01" } },
  "storage": {
    "s3": { "endpoint": "1.2.3.4:9000", "accessKey": "...", "secretKey": "...", "bucket": "velero", "useSSL": false }
  }
}
```

- EKS/GKE/NCP → 자동으로 broker-exec 로 변환
- Azure AKS → 임베드 kubeconfig 그대로

### 7.2 kubeconfig 직접 전달 방식 (기존, 하위 호환)

```json
{ "sourceCluster": { "kubeconfig": "<base64 또는 평문 kubeconfig>" } }
```

두 방식 모두 동일한 해석 경로(`ResolveKubeconfig`)를 통과합니다. 둘 중 하나는 반드시 있어야 하며, 없으면 검증에서 거부됩니다.

---

## 8. 검증 결과

- `go build ./...`, `go vet`, `gofmt`, 단위 테스트 모두 통과.
- **end-to-end 실증**: 코드가 생성하는 것과 동일한 broker-exec kubeconfig 로, AWS CLI·자격증명 **0개** 상태에서 라이브 EKS 클러스터에 접근 성공.

```text
$ kubectl --kubeconfig broker-kc.yaml get namespaces
NAME              STATUS   AGE
default           Active   3h37m
kube-node-lease   Active   3h37m
kube-public       Active   3h37m
kube-system       Active   3h37m
```

즉 `sh` + `curl` + `jq` + cb-tumblebug basic-auth 만으로 EKS API 인증이 동작함을 확인했습니다.

---

## 9. 알려진 제약 / 후속 과제

- **토큰 만료 처리**: cb-tumblebug `/token` 응답에는 `status.expirationTimestamp` 가 없습니다(현재 `status.token` 만 존재). 이 경우 client-go 는 토큰을 미리 갱신하지 못하고 **401 을 받은 뒤 exec 를 재실행**하여 self-heal 합니다(요청 1회 실패 후 복구). 매끄럽게 하려면 cb-tumblebug/cb-spider 가 응답에 만료 시각(EKS STS 기준 약 15분)을 채워주는 것이 좋습니다 — 이는 cm-grasshopper 범위 밖의 개선입니다.
- **자격증명 노출**: broker-exec kubeconfig 에는 cb-tumblebug basic-auth 자격증명이 평문으로 들어갑니다. 이는 기존에 cm-grasshopper 가 TB 와 통신하던 방식과 동일한 수준이며, kubeconfig 가 저장/전달될 때의 보호는 운영 환경에서 관리해야 합니다.
- **NHN** 등 기타 CSP 는 kubeconfig 형식을 추가 확인 후 동일 패턴 적용 가능 여부를 판단해야 합니다.

---

## 10. velero 백업/마이그레이션 구현자를 위한 테스트 가이드

velero 백업/복원 로직을 구현·검증하는 사람이 이 kubeconfig 해석 위에서 바로 테스트할 수 있도록 정리합니다.

### 10.1 전제 조건

- cm-grasshopper 설정에 cb-tumblebug 접속 정보가 채워져 있어야 함:
  ```yaml
  cm-grasshopper:
    tumblebug:
      server_address: <TB 주소>
      server_port: "1323"
      username: <TB 계정>
      password: <TB 비밀번호>
  ```
- 실행 환경(컨테이너/로컬)에 **`sh` + `curl` + `jq`** 존재. 컨테이너는 이미 Dockerfile 에 포함됨. **로컬 바이너리로 직접 돌릴 때는 `jq` 를 별도 설치**해야 EKS/GKE/NCP 가 동작함.
- 실행 환경에서 **cb-tumblebug 와 대상 클러스터 API 서버, S3 엔드포인트** 에 네트워크 도달 가능해야 함.

### 10.2 테스트용 라이브 클러스터 (참고)

| 용도 | namespaceId | k8sClusterId | CSP | kubeconfig 유형 |
|---|---|---|---|---|
| exec-plugin 검증 | `testns01` | `ekstest01` | AWS EKS | broker-exec 로 재작성됨 |
| 임베드 검증 | `testns01` | `k8stest01` | Azure AKS | 원본 그대로 |

> 참고: `ekstest01` 은 컨트롤플레인만 Active 이고 노드그룹이 비어 있을 수 있음(AWS 는 클러스터 생성 후 노드그룹을 별도 추가). 실제 워크로드 백업/복원까지 보려면 노드그룹을 추가해 워커 노드를 띄워야 함.

### 10.3 단계별 테스트

**0단계 — kubeconfig 해석만 단독 확인 (가장 빠른 사전 검증)**

velero 를 거치지 않고, broker-exec 가 실제로 클러스터 인증에 성공하는지 먼저 확인합니다. (실증에 사용한 방법)

```bash
TB="http://<TB>:1323"; NS="testns01"; CID="ekstest01"
SERVER=$(curl -fsS -u <user>:<pass> "$TB/tumblebug/ns/$NS/k8sCluster/$CID/kubeconfig" | jq -r '.kubeconfig' | awk '/server:/{print $2}')
CA=$(curl -fsS -u <user>:<pass> "$TB/tumblebug/ns/$NS/k8sCluster/$CID/kubeconfig" | jq -r '.kubeconfig' | awk '/certificate-authority-data:/{print $2}')
cat > broker-kc.yaml <<EOF
apiVersion: v1
kind: Config
clusters: [{cluster: {server: $SERVER, certificate-authority-data: $CA}, name: c}]
contexts: [{context: {cluster: c, user: u}, name: c}]
current-context: c
users:
- name: u
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      interactiveMode: Never
      command: sh
      args: ["-c", "curl -fsS -u '<user>:<pass>' '$TB/tumblebug/ns/$NS/k8sCluster/$CID/token' | jq -ce .execCredential"]
EOF
kubectl --kubeconfig broker-kc.yaml get ns   # namespace 목록이 나오면 성공
```

**1단계 — velero 설치 (source/target 각각)**

```
POST /grasshopper/velero/source/install
POST /grasshopper/velero/target/install
```
본문에 `sourceCluster`/`targetCluster` 를 TB 참조로, `storage.s3` 를 채워 보냄. → velero 파드가 뜨고 BackupStorageLocation 이 Available 인지 확인.

**2단계 — 사전 점검**

```
POST /grasshopper/velero/migration/precheck
```
→ 두 클러스터 접근 / S3 연결 / 네임스페이스·스토리지클래스 점검 결과 확인.

**3단계 — 마이그레이션 실행**

```
POST /grasshopper/velero/migration/execute
```
→ 비동기 Job 으로 backup → S3 sync → restore 진행. Job 로그/상태로 진척 추적.

### 10.4 확인 포인트 / 트러블슈팅

| 증상 | 원인 후보 | 확인 |
|---|---|---|
| `executable jq not found` / exec 실패 | 실행 환경에 `jq` 없음 | 컨테이너/로컬에 `jq` 설치 확인 |
| `couldn't get current server API group list` 류 인증 실패 | TB `/token` 도달 불가 또는 자격증명 오류 | 0단계 스크립트로 단독 재현, `curl … /token` 직접 호출 |
| 처음엔 되다가 15분쯤 뒤 401 한 번 | 토큰 만료 (expirationTimestamp 부재) | 정상 동작 — client-go 가 exec 재실행으로 self-heal. 9장 참고 |
| Azure 는 되는데 EKS 만 실패 | exec 경로(jq/curl/네트워크) 문제 | 9장·10.1 의 exec 전제조건 재확인 |
| S3 관련 실패 | 엔드포인트 형식/도달성 | `S3Access.endpoint` 는 scheme·path 없이 `host[:port]`, `useSSL` 별도 |

> **빠른 격리법**: 문제가 "kubeconfig 인증"인지 "velero 로직"인지 구분하려면 항상 **0단계(kubectl 단독)** 를 먼저 돌려보세요. 0단계가 되면 인증 계층은 정상이고 velero 쪽을 보면 됩니다.
