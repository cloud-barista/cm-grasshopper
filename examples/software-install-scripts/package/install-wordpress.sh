#!/bin/bash

## WordPress Install Script by ish
## Working with Ubuntu 22.04

if [ "$EUID" -ne 0 ]; then
  echo "[!] Please run as root or use sudo!"
  exit 1
fi

apt update -y

# Install NGINX
apt install -y nginx

# Install MariaDB
apt install -y mariadb-server

# Install PHP8
apt install -y php8.1-fpm php8.1-mysql php8.1-curl php8.1-gd php8.1-intl php8.1-mbstring php8.1-soap php8.1-xml php8.1-xmlrpc php8.1-zip

# Create Wordpress database
mysql -uroot -e "create database if not exists wordpress_db character set utf8mb4 collate utf8mb4_general_ci";
mysql -uroot -e "create user 'wp_user'@'localhost' identified by 'qwe1212!Q'";
mysql -uroot -e "grant all privileges on wordpress_db.* TO 'wp_user'@'localhost'";

# Download and Extract WordPress
wget https://wordpress.org/latest.tar.gz -O /tmp/wordpress_latest.tar.gz
tar xvf /tmp/wordpress_latest.tar.gz -C /var/www/html/
chown -R www-data:www-data /var/www/html/wordpress

# Write WordPress site file
cat << EOF > /etc/nginx/sites-available/default
server {
    listen 80;
    server_name your_domain.com www.your_domain.com;
    root /var/www/html/wordpress;
    index index.php;

    location / {
        try_files $uri $uri/ /index.php?$args;
    }

    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:/var/run/php/php8.1-fpm.sock;
        fastcgi_index index.php;
    }

    location ~ /\.ht {
        deny all;
    }

    location = /favicon.ico {
        log_not_found off;
        access_log off;
    }

    location = /robots.txt {
        log_not_found off;
        access_log off;
        allow all;
    }

    location ~* \.(css|gif|ico|jpeg|jpg|js|png)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
EOF

# Configure WordPress config
cp -pf /var/www/html/wordpress/wp-config-sample.php /var/www/html/wordpress/wp-config.php
sed -i "s/define([[:space:]]*'DB_NAME'[[:space:]]*,[[:space:]]*'database_name_here'[[:space:]]*);/define('DB_NAME', 'wordpress_db');/" /var/www/html/wordpress/wp-config.php
sed -i "s/define([[:space:]]*'DB_USER'[[:space:]]*,[[:space:]]*'username_here'[[:space:]]*);/define('DB_USER', 'wp_user');/" /var/www/html/wordpress/wp-config.php
sed -i "s/define([[:space:]]*'DB_PASSWORD'[[:space:]]*,[[:space:]]*'password_here'[[:space:]]*);/define('DB_PASSWORD', 'qwe1212!Q');/" /var/www/html/wordpress/wp-config.php
sed -i "s/define([[:space:]]*'DB_HOST'[[:space:]]*,[[:space:]]*'localhost'[[:space:]]*);/define('DB_HOST', 'localhost');/" /var/www/html/wordpress/wp-config.php

SALT_KEYS=$(curl -s https://api.wordpress.org/secret-key/1.1/salt/)

echo $SALT_KEYS

AUTH_KEY=$(echo "$SALT_KEYS" | grep "AUTH_KEY" | grep -v "SECURE")
SECURE_AUTH_KEY=$(echo "$SALT_KEYS" | grep "SECURE_AUTH_KEY")
LOGGED_IN_KEY=$(echo "$SALT_KEYS" | grep "LOGGED_IN_KEY")
NONCE_KEY=$(echo "$SALT_KEYS" | grep "NONCE_KEY")
AUTH_SALT=$(echo "$SALT_KEYS" | grep "AUTH_SALT" | grep -v "SECURE")
SECURE_AUTH_SALT=$(echo "$SALT_KEYS" | grep "SECURE_AUTH_SALT")
LOGGED_IN_SALT=$(echo "$SALT_KEYS" | grep "LOGGED_IN_SALT")
NONCE_SALT=$(echo "$SALT_KEYS" | grep "NONCE_SALT")

sed -i "/define([[:space:]]*'AUTH_KEY'/c\\$AUTH_KEY" /var/www/html/wordpress/wp-config.php
sed -i "/define([[:space:]]*'SECURE_AUTH_KEY'/c\\$SECURE_AUTH_KEY" /var/www/html/wordpress/wp-config.php
sed -i "/define([[:space:]]*'LOGGED_IN_KEY'/c\\$LOGGED_IN_KEY" /var/www/html/wordpress/wp-config.php
sed -i "/define([[:space:]]*'NONCE_KEY'/c\\$NONCE_KEY" /var/www/html/wordpress/wp-config.php
sed -i "/define([[:space:]]*'AUTH_SALT'/c\\$AUTH_SALT" /var/www/html/wordpress/wp-config.php
sed -i "/define([[:space:]]*'SECURE_AUTH_SALT'/c\\$SECURE_AUTH_SALT" /var/www/html/wordpress/wp-config.php
sed -i "/define([[:space:]]*'LOGGED_IN_SALT'/c\\$LOGGED_IN_SALT" /var/www/html/wordpress/wp-config.php
sed -i "/define([[:space:]]*'NONCE_SALT'/c\\$NONCE_SALT" /var/www/html/wordpress/wp-config.php

# Configure PHP
sed -i 's/^memory_limit[[:space:]]*=.*/memory_limit = 256M/' /etc/php/8.1/fpm/php.ini
sed -i 's/^upload_max_filesize[[:space:]]*=.*/upload_max_filesize = 1024M/' /etc/php/8.1/fpm/php.ini
sed -i 's/^post_max_size[[:space:]]*=.*/post_max_size = 1024M/' /etc/php/8.1/fpm/php.ini
sed -i 's/^max_execution_time[[:space:]]*=.*/max_execution_time = 300/' /etc/php/8.1/fpm/php.ini

# Restart php-fpm
systemctl restart php8.1-fpm

# Restart NGINX
systemctl restart nginx
