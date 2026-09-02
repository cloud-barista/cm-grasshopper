# 쿠버네티스 마이그레이션 현황과 남은 작업

> 기준: `main` (2026-09-02)
>
> 이 문서는 cm-grasshopper 의 쿠버네티스 이관이 **지금 어디까지 되고 어디부터 안 되는지**를 정리한다.
> 설계와 사용법은 [`k8s_cluster_migration.md`](k8s_cluster_migration.md) 를 본다.
>
> 근거는 현재 코드를 읽은 것이다. 실행해 확인한 항목은 따로 표시했다.

---

## 1. 경로가 두 갈래다

cm-grasshopper 에서 "쿠버네티스"라는 말은 서로 다른 두 가지를 가리킨다. 섞어서 보면 "된다/안 된다"가 계속 어긋나므로 먼저 나눈다.

| | (A) Velero 클러스터 이관 | (B) 소프트웨어 이관 안의 Kubernetes 타입 |
|---|---|---|
| 대상 | CB-Tumblebug 이 배포한 관리형 클러스터 | kubeadm self-managed 클러스터 |
| 진입점 | `POST /grasshopper/velero/*`, `/grasshopper/velero/migration/*` | `POST /grasshopper/software/migrate` |
| 소스 정보 | 요청 본문의 `ClusterAccess` | cm-honeybee refined 모델 |
| 실행 수단 | Velero CR 생성 (클러스터에 파일을 쓰지 않는다) | SSH + Ansible |
| 현재 상태 | **동작한다** | **연결되지 않았다** |

---

## 2. (A) Velero 이관 - 동작한다

라우트는 `pkg/api/rest/route/velero.go:11-31` 과 `pkg/api/rest/route/job.go:11-15` 에서 등록된다.

| 기능 | 라우트 |
|---|---|
| 헬스 체크 | `POST /velero/{role}/health` |
| 설치·업그레이드 | `POST /velero/{role}/install` |
| 백업 | 생성 · 목록 · 조회 · 검증 · 삭제 |
| 복원 | 생성 · 목록 · 조회 · 검증 · 삭제 |
| 사전점검 | `POST /velero/migration/precheck` (동기) |
| 실행 | `POST /velero/migration/execute` (비동기 Job) |
| Job | `GET /job/status/{jobId}` · `/job/status` · `/job/log/{jobId}` |

### 호출 순서

**install → precheck → execute** 순서로 부른다. `ExecuteMigrationAsync` 는 Velero 를 설치하지 않는다. `Precheck` 가 `HealthCheck` 를 거치는데(`lib/k8s/velero/service.go:154-161`) 이는 Velero CRD 가 없으면 실패하므로, install 을 먼저 부르지 않으면 execute 는 반드시 실패한다.

### 판정할 때 주의할 것

- **Job 이 `completed` 라고 설치가 온전한 것은 아니다.** `BackupStorageLocation` 이 `Unavailable` 일 수 있으므로 `health` 로 따로 확인한다.
- **복원도 Job 상태로 판정하지 않는다.** 대상 클러스터에서 리소스·StorageClass·볼륨 데이터를 직접 대조한다.

---

## 3. (B) 소프트웨어 이관의 Kubernetes 타입 - 연결되지 않았다

`SoftwareType` 이 `kubernetes` 인 항목은 **에러 없이 조용히 무시된다.** 응답은 200 이다.

| 요청 | 결과 |
|---|---|
| `POST /software/migration_list` 본문에 `softwares.kubernetes[]` | 응답의 `migration_list.kubernetes` 가 빈 값 |
| `POST /software/migrate` 본문에 `migration_list.kubernetes[]` | 상태 레코드가 생성되지 않고 실행 루프도 돌지 않는다 |

끊긴 곳이 세 군데다. 하나만 고쳐서는 효과가 없다.

### 3-1. migration_list 에 변환 함수가 없다

`lib/software/migrationList.go:388-390` 의 `MakeMigrationListRes` 는 Binaries / Packages / Containers 셋만 채운다. Kubernetes 를 채우는 함수가 존재하지 않는다.

### 3-2. 상태 레코드 생성 루프가 없다 (`lib/software/install.go:278`)

`PrepareSoftwareMigration` 안에서 Binaries / Packages / Containers 는 각각 루프를 돌며 `SoftwareMigrationStatus` 를 만들어 저장하는데(`:172`, `:229`, `:268`), Kubernetes 루프만 없다. 3-1 때문에 이 자리를 채워도 입력이 항상 비어 있다.

### 3-3. 실행부가 비어 있다 (`lib/software/install.go:675`)

```go
case softwaremodel.SoftwareTypeKubernetes:
    kubernetes := getKubernetes(execution, ms)
    if kubernetes == nil {
        continue
    }
    // TODO - Kubernetes Migration
```

항목을 찾은 뒤 아무것도 하지 않고 case 를 빠져나간다. 상태 레코드가 존재했다면 그 행은 `ready` 로 남고 전체 실행은 `finished` 가 된다. 다만 3-2 때문에 실제로는 이 case 에 도달하지 않는다.

---

## 4. 먼저 정해야 하는 것 - 모델 계약

코드를 채우기 전에 결정이 필요하다. 이것이 실제 병목이다.

### 4-1. 대상 클러스터를 가리킬 방법이 없다

- `smdl.KubernetesMigrationInfo` 는 소스 kubeconfig 만 갖는다 (`smdl/softwaremodel.go:251-259`).
- 소프트웨어 이관의 target 은 `model.Target{NamespaceID, InfraID, NodeID}` 즉 **VM 노드**다 (`lib/software/install.go:159-163`).
- Velero 쪽 클러스터 참조는 `TumblebugK8sRef{NamespaceID, K8sClusterID}` (`pkg/api/rest/model/common/k8s.go:17`) 로 모양이 다르다.
- 이 둘을 잇는 매핑 규칙이 정의되어 있지 않다.

### 4-2. `KubernetesVelero` 가 소비되지 않는다

`smdl/softwaremodel.go:242-249` 의 필드(`Provider`, `Plugins`, `Bucket`, `SecretFile`, `BackupLocationConfig`, `Features`)는 Velero CLI 플래그 형태이고, Velero 서비스가 받는 `commonmodel.S3Access`(`endpoint`, `accessKey`, `secretKey`, `bucket`, `region`, `prefix`, `useSSL`) 와 형태가 다르다.

이 타입은 **Go 코드 어디에서도 읽히지 않는다.** 선언과 Swagger 생성물이 전부다.

### 4-3. 백업 범위를 정할 근거가 없다

`KubernetesMigrationInfo.Resources` 는 `map[string]interface{}` 이고 스키마가 정의되어 있지 않다. 어떤 네임스페이스와 리소스를 백업 대상으로 잡을지 결정할 수 없다.

### 4-4. 상태 모델이 서로 다르다

Velero 흐름은 비동기 Job(`lib/k8s/velero/service.go:443`, pending → processing → completed/failed)이고, 소프트웨어 흐름은 동기 재시도 루프(`lib/software/install.go:373`)다. 3-3 을 채울 때 `MigrateSoftware` 가 Job 완료를 기다릴지, job id 를 상태 레코드에 남기고 비동기로 둘지 정해야 한다.

---

## 5. 남은 작업

앞의 넷은 서로 의존한다. 순서를 바꾸면 죽은 코드를 만든다.

1. **`smdl.KubernetesMigrationInfo` 에 대상 클러스터 참조와 오브젝트 스토리지 접속 정보를 표현할 필드를 정한다.** 4절의 결정이 여기 모인다.
2. **`MakeMigrationListRes` 에 Kubernetes 변환을 추가한다** (`lib/software/migrationList.go:388-390` 옆).
3. **`install.go:278` 에 Kubernetes 상태 레코드 생성 루프를 추가한다.** 나머지 셋과 형태가 같다.
4. **`install.go:675` 를 Velero 서비스 호출로 채운다.** 4-4 의 결정을 반영한다.

### 위와 독립인 항목

| 항목 | 내용 |
|---|---|
| `:role` 검증 | `getClusterFromRole`(`pkg/api/rest/controller/velero.go:20-25`) 이 `source` 외 모든 값을 target 으로 처리한다. 오타가 조용히 대상 클러스터에 설치·조회를 수행한다 |
| execute 안내 | Velero 미설치 시 `Precheck` 안의 `HealthCheck` 가 원시 Kubernetes API 에러로 실패해, "Velero 를 먼저 설치하라"는 신호가 되지 못한다 |
| 기동 조건 | `lib/config/cm-grasshopper.go:62` 가 `checkSoftwareMigrationConfig` 를 무조건 호출한다. 소프트웨어 이관용 경로(temp / log / playbook)가 없으면 Kubernetes 전용 배포도 기동에 실패한다 |
| 오프라인 설치 | `lib/k8s/installer/velero.go:31` 이 install 마다 Helm 차트를 원격에서 받는다. 폐쇄망을 지원하려면 차트 임베드나 미러 설정이 필요하다 |
| 테스트 | `pkg/api/rest/controller/velero.go` 와 `lib/software` 에 테스트 파일이 없다 |

---

## 6. 알려진 제약 (계속 유효)

[`k8s_cluster_migration.md`](k8s_cluster_migration.md) §9 의 항목 중 아직 해소되지 않은 것.

| 제약 | 내용 |
|---|---|
| 토큰 만료 | CB-Tumblebug 토큰 응답에 `expirationTimestamp` 가 없어 선제 갱신이 안 된다. 401 을 한 번 받은 뒤 exec 재실행으로 복구하는 구조다 |
| 자격증명 노출 | broker-exec 방식으로 재작성한 kubeconfig 의 exec 인자에 인증 정보가 평문으로 들어간다 (`lib/k8s/tumblebug/resolve.go:104-105`) |
| 오브젝트 스토리지 도달성 | Velero 파드가 클러스터 안에서 접속하므로, 클러스터에서 닿지 않는 스토리지는 설치가 성공해도 BSL 이 `Unavailable` 이 된다. 다만 `precheck` 를 거치면 명시적 오류로 드러난다 |
| CSP 범위 | exec-plugin 재작성은 `authInfo.Exec` 가 있는 모든 사용자에 일괄 적용된다. CSP 별 분기가 없어 확인되지 않은 CSP 는 별도 검증이 필요하다 |

### 해소된 것

| 항목 | 해소 방식 |
|---|---|
| source/target 저장소 위치 불일치 | 백업 완료 직후 양쪽 BSL 의 버킷·접두사를 비교해 즉시 실패시킨다 (`lib/k8s/velero/service.go:1479-1534`). 이전에는 대상이 백업을 발견하지 못한 채 타임아웃까지 갔다 |
| BSL region 하드코딩 | API 에 `S3Access.Region` 을 추가하고 기본값을 상수로 뺐다 |
| Helm values 와 controller-runtime 경로의 BSL spec 불일치 | 양쪽 모두 공통 빌더를 거치도록 통일했다 |
| 토큰 요청 버스트 | 재작성한 exec 명령에 재시도 옵션을 넣어 흡수한다 |

---

## 7. 문서 갱신이 필요한 곳

- [`k8s_cluster_migration.md`](k8s_cluster_migration.md) §8 · §10.3 의 재현 스크립트가 현재 코드가 생성하는 exec 명령과 다르다. 재시도 옵션이 반영되지 않았다.
- 같은 문서 §9 의 "설치는 성공하는데 BSL 만 조용히 `Unavailable`" 서술은 실제보다 비관적이다. `precheck` 가 이를 명시적 오류로 올린다.
- 같은 문서 §9 에 BSL 위치 불일치 조기 실패가 반영되지 않았다. §7.1 에만 있다.
- 설정 예시에 등장하는 `features` 토글(`software_migration`, `k8s_migration`)은 현재 코드에 없다. 두 서브시스템은 항상 활성화되고, 소프트웨어 이관의 런타임 의존성은 없어도 경고만 남기고 기동을 계속한다 (`cmd/cm-grasshopper/main.go:87-91`).
