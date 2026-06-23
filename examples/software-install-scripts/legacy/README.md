# Legacy 소프트웨어 설치 스크립트 (바이너리 마이그레이션 Source 셋업)

이 스크립트들은 소프트웨어를 **OS 패키지 매니저가 아닌 압축파일(tarball)로 수동
설치**하고 **systemd 서비스로 실행**합니다. 이렇게 하면 cm-grasshopper의 Legacy
(바이너리) 소프트웨어 마이그레이션을 테스트할 수 있는 현실적인 *Source* 호스트가
됩니다. honeybee가 실행 방식(launch type)을 수집하고, grasshopper가 대상(Target)
에서 그 시작 메커니즘을 그대로 재현하는 흐름을 검증할 수 있습니다.

Ubuntu 22.04 / 24.04에서 테스트되었습니다. root 권한으로(또는 `sudo`로) 실행하세요.

| 스크립트 | 소프트웨어 | 설치 경로 | 서비스 | 접속 주소 |
|--------|----------|--------------|---------|-----|
| `apache/install-apache-legacy.sh` | Apache HTTP Server (최신 2.4, 소스 빌드) | `/usr/local/apache2` | `httpd.service` | http://localhost/ |
| `tomcat/install-tomcat-legacy.sh` | JDK 21 (Temurin) + Apache Tomcat (최신 GA) | `/opt/jdk`, `/opt/tomcat` | `tomcat.service` | http://localhost:8080/ |

각 스크립트는 최신 버전을 자동으로 다운로드하고, 수동으로 설정한 뒤, 마지막에
서비스가 정상 구동되었음을 확인할 수 있는 상태 페이지를 제공합니다.

## 사용 방법

```bash
sudo ./apache/install-apache-legacy.sh   # Apache, 80번 포트
sudo ./tomcat/install-tomcat-legacy.sh   # Tomcat, 8080번 포트 (JDK 21도 함께 설치)
```

## 각 스크립트가 하는 일

### `install-apache-legacy.sh`

1. 빌드 도구(`build-essential`, `libpcre2-dev` 등)를 설치합니다. *빌드 도구는
   apt로 받지만 Apache 바이너리 자체는 소스에서 직접 빌드*하므로 `dpkg`로 추적되지
   않는 Legacy 설치 형태가 됩니다.
2. Apache 미러에서 최신 httpd 2.4 소스 tarball과 APR / APR-util을 자동 탐지하여
   다운로드합니다.
3. APR / APR-util을 httpd의 `srclib`에 넣고 `./configure → make → make install`로
   `/usr/local/apache2`에 수동 설치합니다.
4. Apache가 정상 동작 중임을 보여주는 한글 상태 페이지(`htdocs/index.html`)를
   작성합니다.
5. `httpd.service` systemd 유닛을 등록·기동하고, `curl`로 80번 포트 응답을
   확인합니다.

### `install-tomcat-legacy.sh`

1. CPU 아키텍처(x64 / aarch64)를 감지합니다.
2. [Adoptium API](https://api.adoptium.net)에서 **JDK 21**(Eclipse Temurin)을
   tarball로 받아 `/opt`에 풀고 `/opt/jdk`로 심볼릭 링크합니다. `JAVA_HOME`은
   `/etc/profile.d/jdk.sh`로 시스템 전역에 설정합니다.
3. Apache 미러에서 최신 **Tomcat 11** GA 버전을 자동 탐지하여 tarball로 받아
   `/opt`에 풀고 `/opt/tomcat`으로 심볼릭 링크합니다.
4. 전용 서비스 계정(`tomcat`)을 만들고, 기본 ROOT 앱을 Tomcat이 정상 동작 중임을
   보여주는 한글 상태 페이지(`webapps/ROOT/index.jsp` — 현재 시간, 서버 정보, JVM
   버전 표시)로 교체합니다.
5. `JAVA_HOME`, `CATALINA_HOME` 등을 지정한 `tomcat.service` systemd 유닛을
   등록·기동하고, `curl`로 8080번 포트 응답을 확인합니다.

## 참고 사항

- **Apache**는 공식 소스 tarball(APR / APR-util 포함)에서 컴파일합니다.
- **Tomcat**은 JDK가 필요하므로 JDK 21을 함께 설치합니다. JDK와 Tomcat 모두
  심볼릭 링크(`/opt/jdk`, `/opt/tomcat`)로 연결되어 있어, 서비스 유닛을 건드리지
  않고도 버전을 교체할 수 있습니다.
- **제거 방법:** `systemctl disable --now httpd`(또는 `tomcat`) 실행 →
  `/etc/systemd/system/`의 유닛 파일 삭제 → 설치 디렉터리 삭제.
