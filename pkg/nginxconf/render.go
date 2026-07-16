// Package nginxconf provides shared Nginx configuration rendering for both
// the local binary driver and the remote Ansible deploy path.
// All config content is generated here to guarantee consistency.
package nginxconf

import "fmt"

// Config holds the parameters needed to render all Nginx config files.
// Both the binary driver (local install) and the Ansible playbook generator
// populate this struct from Viper config, ensuring a single source of truth.
type Config struct {
	InstallPath     string
	WWWRoot         string
	SSLRoot         string
	WWWLogsRoot     string
	RunUser         string
	RunGroup        string
	SystemdUnitPath string
	LogrotatePath   string
}

// RenderBaseConfig returns the content for nginx.conf (main config file).
func RenderBaseConfig(c Config) string {
	return fmt.Sprintf(`user %s %s;
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
`, c.RunUser, c.RunGroup, c.WWWLogsRoot, c.WWWLogsRoot, c.WWWRoot)
}

// RenderProxyConfig returns the content for proxy.conf (shared proxy settings).
func RenderProxyConfig() string {
	return `proxy_connect_timeout 300s;
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
}

// RenderSystemdUnit returns the content for the nginx systemd service unit file.
func RenderSystemdUnit(c Config) string {
	nginxBin := c.InstallPath + "/sbin/nginx"
	nginxConf := c.InstallPath + "/conf/nginx.conf"
	return fmt.Sprintf(`[Unit]
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
`, nginxBin, nginxConf, nginxBin, nginxConf)
}

// RenderLogrotate returns the content for /etc/logrotate.d/nginx.
func RenderLogrotate(c Config) string {
	return fmt.Sprintf(`%s/*nginx.log {
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
`, c.WWWLogsRoot)
}
