package nginxconf

// baseConfigTemplate is the raw template for the main nginx.conf.
const baseConfigTemplate = `user %s %s;
worker_processes auto;

error_log %s/error_nginx.log crit;
pid /var/run/nginx.pid;
worker_rlimit_nofile 51200;

events {
  use epoll;
  worker_connections 51200;
  multi_accept on;
}

http {
  include mime.types;
  default_type application/octet-stream;
  server_names_hash_bucket_size 128;
  client_header_buffer_size 32k;
  large_client_header_buffers 4 32k;
  client_max_body_size 1024m;
  client_body_buffer_size 10m;
  sendfile on;
  tcp_nopush on;
  keepalive_timeout 120;
  server_tokens off;
  tcp_nodelay on;

  gzip on;
  gzip_buffers 16 8k;
  gzip_comp_level 6;
  gzip_http_version 1.1;
  gzip_min_length 256;
  gzip_proxied any;
  gzip_vary on;
  gzip_types text/xml application/xml application/atom+xml application/rss+xml application/xhtml+xml image/svg+xml text/javascript application/javascript application/x-javascript text/x-json application/json text/css text/plain image/x-icon;
  gzip_disable "MSIE [1-6]\.(?!.*SV1)";

  map $http_upgrade $connection_upgrade {
    default upgrade;
    '' close;
  }

  log_format json escape=json '{"@timestamp":"$time_iso8601","server_addr":"$server_addr","remote_addr":"$remote_addr","scheme":"$scheme","request_method":"$request_method","request_uri":"$request_uri","request_time":$request_time,"body_bytes_sent":$body_bytes_sent,"status":"$status","host":"$host","http_referer":"$http_referer","http_user_agent":"$http_user_agent"}';

  server {
    listen 80;
    server_name _;
    access_log %s/access_nginx.log combined;
    root %s/default;
    index index.html index.htm;

    location /nginx_status {
      stub_status on;
      access_log off;
      allow 127.0.0.1;
      deny all;
    }

    location ~ .*\.(gif|jpg|jpeg|png|bmp|swf|flv|mp4|ico)$ {
      expires 30d;
      access_log off;
    }

    location ~ .*\.(js|css)?$ {
      expires 7d;
      access_log off;
    }

    location ~ ^/(\.user.ini|\.ht|\.git|\.svn|\.project|LICENSE|README.md) {
      deny all;
    }

    location /.well-known {
      allow all;
    }
  }

  include vhost/*.conf;
}
`

// proxyConfigTemplate is the raw template for proxy.conf.
const proxyConfigTemplate = `proxy_connect_timeout 300s;
proxy_send_timeout 900;
proxy_read_timeout 900;
proxy_buffer_size 32k;
proxy_buffers 4 64k;
proxy_busy_buffers_size 128k;
proxy_redirect off;
proxy_hide_header Vary;
proxy_set_header Accept-Encoding '';
proxy_set_header Referer $http_referer;
proxy_set_header Cookie $http_cookie;
proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;
`

// systemdUnitTemplate is the raw template for systemd service unit.
const systemdUnitTemplate = `[Unit]
Description=nginx - high performance web server
Documentation=http://nginx.org/en/docs/
After=network.target

[Service]
Type=forking
PIDFile=/var/run/nginx.pid
ExecStartPost=/bin/sleep 0.1
ExecStartPre=%s -t -c %s
ExecStart=%s -c %s
ExecReload=/bin/kill -s HUP $MAINPID
ExecStop=/bin/kill -s QUIT $MAINPID
TimeoutStartSec=120
LimitNOFILE=1000000
LimitNPROC=1000000
LimitCORE=1000000

[Install]
WantedBy=multi-user.target
`

// logrotateTemplate is the raw template for logrotate.
const logrotateTemplate = `%s/*nginx.log {
  daily
  rotate 5
  missingok
  dateext
  compress
  notifempty
  sharedscripts
  postrotate
    [ -e /var/run/nginx.pid ] && kill -USR1 $(cat /var/run/nginx.pid)
  endscript
}
`

// vhostHTTPProxyTemplate is the raw template for an HTTP proxy vhost.
const vhostHTTPProxyTemplate = `server {
    listen 80;
    server_name %s;

    access_log %s json;
    error_log %s error;

    location /.well-known/acme-challenge/ {
        allow all;
        root %s;
    }

    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    location / {
        proxy_pass %s;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Port $server_port;

        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;

        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        proxy_buffer_size 128k;
        proxy_buffers 4 256k;
        proxy_busy_buffers_size 256k;
    }

    location ~ /\. {
        deny all;
    }
}
`

// vhostHTTPStaticTemplate is the raw template for an HTTP static vhost.
const vhostHTTPStaticTemplate = `server {
    listen 80;
    server_name %s;

    access_log %s json;
    error_log %s error;

    root %s;
    index index.html index.htm;

    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location ~* \.(gif|jpg|jpeg|png|bmp|swf|flv|mp4|ico|svg)$ {
        expires 30d;
        access_log off;
    }

    location ~* \.(js|css)$ {
        expires 7d;
        access_log off;
    }

    location ~ /\. {
        deny all;
    }
}
`

// vhostSSLProxyLocationTemplate is the location block template for SSL proxy vhost.
const vhostSSLProxyLocationTemplate = `    location / {
        proxy_pass %s;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Port $server_port;

        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;

        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        proxy_buffer_size 128k;
        proxy_buffers 4 256k;
        proxy_busy_buffers_size 256k;
    }`

// vhostSSLStaticLocationTemplate is the location block template for SSL static vhost.
const vhostSSLStaticLocationTemplate = `    root %s;
    index index.html index.htm;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location ~* \.(gif|jpg|jpeg|png|bmp|swf|flv|mp4|ico|svg)$ {
        expires 30d;
        access_log off;
    }

    location ~* \.(js|css)$ {
        expires 7d;
        access_log off;
    }`

// vhostSSLTemplate is the raw template for an HTTPS/SSL vhost.
const vhostSSLTemplate = `server {
    listen 80;
    server_name %s;

    location /.well-known/acme-challenge/ {
        allow all;
        root %s;
    }

    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl;
    server_name %s;

    ssl_certificate %s;
    ssl_certificate_key %s;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    access_log %s json;
    error_log %s error;

    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

%s

    location ~ /\. {
        deny all;
    }
}
`
