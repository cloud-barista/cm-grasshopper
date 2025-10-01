#!/bin/bash

## MariaDB Container Install Script by ish
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

mkdir -p mariadb

# Write Docker Compose file
cat << EOF > mariadb/docker-compose.yaml
services:
  mariadb:
    image: mariadb
    container_name: mariadb_compose
    restart: always
    environment:
      - MYSQL_ROOT_PASSWORD=rootpass
      - MYSQL_DATABASE=testdb
      - MYSQL_USER=testuser
      - MYSQL_PASSWORD=testpass
    volumes:
      - mariadb_data:/var/lib/mysql
    networks:
      - compose-network

networks:
  compose-network:
    driver: bridge

volumes:
  mariadb_data:
EOF

cd mariadb
docker compose up -d

echo "Waiting for MariaDB is up..."
cnt=0
while true
do
   (( cnt = "$cnt" + 1 ))
   MARIADB_STATUS=`docker exec -it mariadb_compose mariadb -uroot -prootpass -e"SELECT 1;" > /dev/null 2>&1 || exit 1 ; echo $?`
   if [ "$MARIADB_STATUS" = "0" ]; then
      break
   fi
   if [ "$cnt" = "60" ]; then
      echo "Failed to connect to MariaDB."
      exit 1;
   fi
   sleep 1
done
echo "MariaDB is now ready!"

docker exec -it mariadb_compose mariadb -uroot -prootpass -e "CREATE DATABASE IF NOT EXISTS newdb;"
