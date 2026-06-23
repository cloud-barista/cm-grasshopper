#!/bin/bash

## Apache HTTP Server Legacy (manual/source) Install Script by ish
## Working with Ubuntu 22.04 and Ubuntu 24.04
##
## Downloads the latest Apache httpd source tarball (plus bundled APR / APR-util),
## builds it manually into /usr/local/apache2, registers a systemd service, and
## serves a status page. This mimics a "legacy" software install (not from the
## package manager) so it can be used as a Source for binary migration testing.

set -e

if [ "$EUID" -ne 0 ]; then
  echo "[!] Please run as root or use sudo!"
  exit 1
fi

PREFIX="/usr/local/apache2"
BUILD_DIR="/usr/local/src/apache-build"
APACHE_MIRROR="https://dlcdn.apache.org"

# Install build dependencies (the httpd binary itself is built from source)
apt-get update -y
apt-get install -y build-essential wget curl libpcre2-dev libexpat1-dev

# Detect the latest available versions from the Apache mirror
echo "[*] Detecting latest Apache httpd / APR versions..."
HTTPD_VERSION=$(curl -fsSL "${APACHE_MIRROR}/httpd/" \
  | grep -oE 'httpd-2\.4\.[0-9]+\.tar\.gz' \
  | sed -E 's/httpd-(.*)\.tar\.gz/\1/' | sort -V | uniq | tail -1)
APR_VERSION=$(curl -fsSL "${APACHE_MIRROR}/apr/" \
  | grep -oE 'apr-1\.[0-9]+\.[0-9]+\.tar\.gz' \
  | sed -E 's/apr-(.*)\.tar\.gz/\1/' | sort -V | uniq | tail -1)
APR_UTIL_VERSION=$(curl -fsSL "${APACHE_MIRROR}/apr/" \
  | grep -oE 'apr-util-1\.[0-9]+\.[0-9]+\.tar\.gz' \
  | sed -E 's/apr-util-(.*)\.tar\.gz/\1/' | sort -V | uniq | tail -1)

if [ -z "$HTTPD_VERSION" ] || [ -z "$APR_VERSION" ] || [ -z "$APR_UTIL_VERSION" ]; then
  echo "[!] Failed to detect versions from the mirror."
  exit 1
fi

echo "[*] httpd=${HTTPD_VERSION} apr=${APR_VERSION} apr-util=${APR_UTIL_VERSION}"

# Download source tarballs
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"
cd "$BUILD_DIR"

wget -q "${APACHE_MIRROR}/httpd/httpd-${HTTPD_VERSION}.tar.gz"
wget -q "${APACHE_MIRROR}/apr/apr-${APR_VERSION}.tar.gz"
wget -q "${APACHE_MIRROR}/apr/apr-util-${APR_UTIL_VERSION}.tar.gz"

# Extract and place APR / APR-util inside httpd's srclib so they are built together
tar xf "httpd-${HTTPD_VERSION}.tar.gz"
tar xf "apr-${APR_VERSION}.tar.gz"
tar xf "apr-util-${APR_UTIL_VERSION}.tar.gz"

mv "apr-${APR_VERSION}" "httpd-${HTTPD_VERSION}/srclib/apr"
mv "apr-util-${APR_UTIL_VERSION}" "httpd-${HTTPD_VERSION}/srclib/apr-util"

# Configure, build and install manually
cd "httpd-${HTTPD_VERSION}"
./configure --prefix="$PREFIX" --with-included-apr --enable-so
make -j"$(nproc)"
make install

# Quiet the "could not reliably determine the server's FQDN" warning
if ! grep -q "^ServerName localhost" "${PREFIX}/conf/httpd.conf"; then
  echo "ServerName localhost" >> "${PREFIX}/conf/httpd.conf"
fi

# Write the status page that proves Apache is serving
cat << 'EOF' > "${PREFIX}/htdocs/index.html"
<!DOCTYPE html>
<html lang="ko">
<head>
    <meta charset="UTF-8">
    <title>Apache 테스트</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 50px auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 { color: #333; }
        .info {
            background: #e7f7e9;
            padding: 15px;
            border-radius: 4px;
            margin: 20px 0;
        }
        code { background: #eee; padding: 2px 6px; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🪶 Apache HTTP Server가 정상 작동 중입니다!</h1>
        <div class="info">
            <p><strong>설치 방식:</strong> 소스 빌드 (수동 설치, 패키지 매니저 미사용)</p>
            <p><strong>설치 경로:</strong> <code>/usr/local/apache2</code></p>
            <p><strong>서비스:</strong> systemd (<code>httpd.service</code>)</p>
        </div>
        <p>이 페이지가 보이면 Legacy 마이그레이션 테스트용 Source의 Apache가 성공적으로 구동된 것입니다.</p>
    </div>
</body>
</html>
EOF

# Register a systemd service so the start mechanism is reproducible on migration
cat << 'EOF' > /etc/systemd/system/httpd.service
[Unit]
Description=Apache HTTP Server (source build)
After=network.target

[Service]
Type=forking
ExecStart=/usr/local/apache2/bin/apachectl start
ExecReload=/usr/local/apache2/bin/apachectl graceful
ExecStop=/usr/local/apache2/bin/apachectl stop
PIDFile=/usr/local/apache2/logs/httpd.pid
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable httpd
systemctl restart httpd

# Verify
echo "[*] Waiting for Apache to come up..."
sleep 2
if curl -fsS http://localhost/ | grep -q "Apache HTTP Server"; then
  echo "[+] Apache is serving on http://localhost/ (port 80)"
else
  echo "[!] Apache did not respond as expected. Check: systemctl status httpd"
  exit 1
fi
