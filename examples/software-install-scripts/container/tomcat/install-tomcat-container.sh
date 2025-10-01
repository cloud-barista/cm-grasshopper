#!/bin/bash

## Tomcat Container Install Script by ish
## Working with Ubuntu 22.04 and Ubuntu 24.04

if [ "$EUID" -ne 0 ]; then
  echo "[!] Please run as root or use sudo!"
  exit 1
fi

# Add Docker's official GPG key:
apt-get -y update
apt-get -y install ca-certificates curl
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc

# Add the repository to Apt sources:
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}") stable" | \
  tee /etc/apt/sources.list.d/docker.list > /dev/null
apt-get -y update

# Install Docker
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

mkdir -p tomcat

# Write Docker Compose file
cat << EOF > tomcat/docker-compose.yaml
services:
  tomcat:
    image: tomcat:10.1-jdk17
    container_name: tomcat_compose
    ports:
      - "8080:8080"
    volumes:
      - ./webapps:/usr/local/tomcat/webapps
      - ./logs:/usr/local/tomcat/logs
    environment:
      - CATALINA_OPTS=-Xms512m -Xmx1024m
    networks:
      - compose-network

networks:
  compose-network:
    driver: bridge
EOF

mkdir -p tomcat/webapps/ROOT

# Write index.jsp file
cat << EOF > tomcat/webapps/ROOT/index.jsp
<%@ page contentType="text/html; charset=UTF-8" pageEncoding="UTF-8" %>
<!DOCTYPE html>
<html>
<head>
    <title>Tomcat 테스트</title>
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
        h1 {
            color: #333;
        }
        .info {
            background: #e8f4f8;
            padding: 15px;
            border-radius: 4px;
            margin: 20px 0;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 Tomcat이 정상 작동 중입니다!</h1>

        <div class="info">
            <p><strong>현재 시간:</strong> <%= new java.util.Date() %></p>
            <p><strong>서버 정보:</strong> <%= application.getServerInfo() %></p>
            <p><strong>세션 ID:</strong> <%= session.getId() %></p>
        </div>

        <p>Docker Compose로 실행된 Tomcat 컨테이너가 성공적으로 구동되었습니다.</p>
    </div>
</body>
</html>
EOF

cd tomcat
docker compose up -d
