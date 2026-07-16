package ansible

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"OpsVault/pkg/mysqlconf"
	"OpsVault/pkg/nginxconf"
	"OpsVault/pkg/rabbitmqconf"
	"OpsVault/pkg/redisconf"
	"OpsVault/pkg/versionutil"
)

// PlaybookTemplates contains built-in ansible playbooks.
var PlaybookTemplates = map[string]string{
	"docker": `---
- name: Install and Setup Docker
  hosts: {{ .TargetGroup }}
  become: yes
  tasks:
    - name: Check if Docker is installed
      command: which docker
      register: docker_check
      ignore_errors: yes

    - name: Install dependencies for Docker
      yum:
        name:
          - yum-utils
          - device-mapper-persistent-data
          - lvm2
        state: present
      when: docker_check.rc != 0

    - name: Add Docker CE repository
      get_url:
        url: https://download.docker.com/linux/centos/docker-ce.repo
        dest: /etc/yum.repos.d/docker-ce.repo
      when: docker_check.rc != 0

    - name: Install Docker CE
      yum:
        name: docker-ce
        state: present
      when: docker_check.rc != 0

    - name: Start and enable Docker service
      systemd:
        name: docker
        state: started
        enabled: yes
`,

	"mysql": `---
- name: Deploy MySQL via Docker
  hosts: {{ .TargetGroup }}
  become: yes
  tasks:
    - name: Create MySQL data and conf directories
      file:
        path: "{{ "{{" }} item {{ "}}" }}"
        state: directory
        mode: '0755'
      loop:
        - "{{ .DataRoot }}/mysql/data"
        - "{{ .DataRoot }}/mysql/conf"

    - name: Write my.cnf
      copy:
        dest: "{{ .DataRoot }}/mysql/conf/my.cnf"
        content: |
          {{ .MySQLMyCnf | indent 10 }}
        mode: '0644'

    - name: Create Docker bridge network if not exists
      shell: "docker network inspect {{ .NetworkName }} || docker network create --subnet={{ .CIDR }} {{ .NetworkName }}"
      register: network_create
      changed_when: "'Created' in network_create.stdout"

    - name: Stop and remove existing MySQL container
      shell: "docker rm -f {{ .NamePrefix }}-mysql || true"

    - name: Run MySQL container
      shell: >
        docker run -d
        --name {{ .NamePrefix }}-mysql
        --restart always
        --network {{ .NetworkName }}
        -p {{ .MySQLPort }}:3306
        -v {{ .DataRoot }}/mysql/data:/var/lib/mysql
        -v {{ .DataRoot }}/mysql/conf/my.cnf:/etc/mysql/conf.d/my.cnf
        -e MYSQL_ROOT_PASSWORD={{ .MySQLRootPassword }}
        {{ .MySQLImage }}
`,

	"redis": `---
- name: Deploy Redis via Docker
  hosts: {{ .TargetGroup }}
  become: yes
  tasks:
    - name: Create Redis data and conf directories
      file:
        path: "{{ "{{" }} item {{ "}}" }}"
        state: directory
        mode: '0755'
      loop:
        - "{{ .DataRoot }}/redis/data"
        - "{{ .DataRoot }}/redis/conf"

    - name: Write redis.conf
      copy:
        dest: "{{ .DataRoot }}/redis/conf/redis.conf"
        content: |
          {{ .RedisCnf | indent 10 }}
        mode: '0644'

    - name: Create Docker bridge network if not exists
      shell: "docker network inspect {{ .NetworkName }} || docker network create --subnet={{ .CIDR }} {{ .NetworkName }}"
      register: network_create
      changed_when: "'Created' in network_create.stdout"

    - name: Stop and remove existing Redis container
      shell: "docker rm -f {{ .NamePrefix }}-redis || true"

    - name: Run Redis container
      shell: >
        docker run -d
        --name {{ .NamePrefix }}-redis
        --restart always
        --network {{ .NetworkName }}
        -p {{ .RedisPort }}:6379
        -v {{ .DataRoot }}/redis/data:/data
        -v {{ .DataRoot }}/redis/conf/redis.conf:/usr/local/etc/redis/redis.conf
        {{ .RedisImage }}
        redis-server /usr/local/etc/redis/redis.conf
`,

	"rabbitmq": `---
- name: Deploy RabbitMQ via Docker
  hosts: {{ .TargetGroup }}
  become: yes
  tasks:
    - name: Create RabbitMQ data and conf directories
      file:
        path: "{{ "{{" }} item {{ "}}" }}"
        state: directory
        mode: '0755'
      loop:
        - "{{ .DataRoot }}/rabbitmq/data"
        - "{{ .DataRoot }}/rabbitmq/conf"

    - name: Write rabbitmq.conf
      copy:
        dest: "{{ .DataRoot }}/rabbitmq/conf/rabbitmq.conf"
        content: |
          {{ .RabbitMQConf | indent 10 }}
        mode: '0644'

    - name: Create Docker bridge network if not exists
      shell: "docker network inspect {{ .NetworkName }} || docker network create --subnet={{ .CIDR }} {{ .NetworkName }}"
      register: network_create
      changed_when: "'Created' in network_create.stdout"

    - name: Stop and remove existing RabbitMQ container
      shell: "docker rm -f {{ .NamePrefix }}-rabbitmq || true"

    - name: Run RabbitMQ container
      shell: >
        docker run -d
        --name {{ .NamePrefix }}-rabbitmq
        --restart always
        --network {{ .NetworkName }}
        -p {{ .RabbitMQPort }}:5672
        -p {{ .RabbitMQUIPort }}:15672
        -v {{ .DataRoot }}/rabbitmq/data:/var/lib/rabbitmq
        -v {{ .DataRoot }}/rabbitmq/conf/rabbitmq.conf:/etc/rabbitmq/rabbitmq.conf
        -e RABBITMQ_DEFAULT_USER={{ .RabbitMQUser }}
        -e RABBITMQ_DEFAULT_PASS={{ .RabbitMQPwd }}
        {{ .RabbitMQImage }}
`,

	// nginx is deployed via binary driver (source compile), not Docker.
	"nginx": `---
- name: Deploy Nginx via source compile (Binary Driver)
  hosts: {{ .TargetGroup }}
  become: yes
  vars:
    nginx_version: "{{ .NginxVersion }}"
    pcre_version: "{{ .NginxPCREVersion }}"
    openssl_version: "{{ .NginxOpenSSLVersion }}"
    openssl_url: "{{ .NginxOpenSSLURL }}"
    install_path: "{{ .NginxInstallPath }}"
    source_root: "{{ .NginxSourceRoot }}"
    www_root: "{{ .NginxWWWRoot }}"
    ssl_root: "{{ .NginxSSLRoot }}"
    wwwlogs_root: "{{ .NginxWWWLogsRoot }}"
    run_user: "{{ .NginxRunUser }}"
    run_group: "{{ .NginxRunGroup }}"
    systemd_unit_path: "{{ .NginxSystemdUnitPath }}"
  tasks:
    - name: Install compile dependencies
      yum:
        name:
          - gcc
          - gcc-c++
          - make
          - automake
          - wget
          - tar
          - zlib
          - zlib-devel
          - libxml2
          - libxml2-devel
          - libxslt
          - libxslt-devel
          - gd
          - gd-devel
          - geoip
          - geoip-devel
          - perl
          - perl-devel
          - perl-ExtUtils-Embed
        state: present

    - name: Create Nginx run user/group
      block:
        - name: Create group {{ "{{" }} run_group {{ "}}" }}
          group:
            name: "{{ "{{" }} run_group {{ "}}" }}"
            state: present
        - name: Create user {{ "{{" }} run_user {{ "}}" }}
          user:
            name: "{{ "{{" }} run_user {{ "}}" }}"
            group: "{{ "{{" }} run_group {{ "}}" }}"
            shell: /sbin/nologin
            create_home: no
            state: present

    - name: Create required directories
      file:
        path: "{{ "{{" }} item {{ "}}" }}"
        state: directory
        mode: '0755'
      loop:
        - "{{ "{{" }} source_root {{ "}}" }}"
        - "{{ "{{" }} www_root {{ "}}" }}"
        - "{{ "{{" }} ssl_root {{ "}}" }}"
        - "{{ "{{" }} wwwlogs_root {{ "}}" }}"

    - name: Download PCRE source
      shell: |
        DEST="{{ "{{" }} source_root {{ "}}" }}/pcre-{{ "{{" }} pcre_version {{ "}}" }}.tar.gz"
        if [ -f "$DEST" ] && [ $(stat -c%s "$DEST" 2>/dev/null || stat -f%z "$DEST" 2>/dev/null || echo 0) -gt 1000000 ]; then
          echo "PCRE source already downloaded ($DEST)."
          exit 0
        fi
        for url in "https://sourceforge.net/projects/pcre/files/pcre/{{ "{{" }} pcre_version {{ "}}" }}/pcre-{{ "{{" }} pcre_version {{ "}}" }}.tar.gz/download" "https://mirrors.aliyun.com/macports/distfiles/pcre/pcre-{{ "{{" }} pcre_version {{ "}}" }}.tar.gz"; do
          echo "Downloading PCRE from mirror: $url"
          if curl -f -L -C - --retry 5 --retry-delay 2 --connect-timeout 15 -sS -o "$DEST" "$url" || wget -c --tries=5 --timeout=30 -q -O "$DEST" "$url"; then
            echo "Successfully downloaded PCRE from $url"
            exit 0
          fi
        done
        echo "Failed to download PCRE source from all mirrors" >&2
        exit 1
      args:
        executable: /bin/bash

    - name: Extract PCRE source
      unarchive:
        src: "{{ "{{" }} source_root {{ "}}" }}/pcre-{{ "{{" }} pcre_version {{ "}}" }}.tar.gz"
        dest: "{{ "{{" }} source_root {{ "}}" }}"
        remote_src: yes

    - name: Download OpenSSL source
      shell: |
        DEST="{{ "{{" }} source_root {{ "}}" }}/openssl-{{ "{{" }} openssl_version {{ "}}" }}.tar.gz"
        if [ -f "$DEST" ] && [ $(stat -c%s "$DEST" 2>/dev/null || stat -f%z "$DEST" 2>/dev/null || echo 0) -gt 10000000 ]; then
          echo "OpenSSL source already downloaded ($DEST)."
          exit 0
        fi
        for url in {{ join .NginxOpenSSLURLs " " }}; do
          echo "Downloading OpenSSL from mirror: $url"
          if curl -f -L -C - --retry 5 --retry-delay 2 --connect-timeout 15 -sS -o "$DEST" "$url"; then
            echo "Successfully downloaded OpenSSL from $url"
            exit 0
          elif wget -c --tries=5 --timeout=30 -q -O "$DEST" "$url"; then
            echo "Successfully downloaded OpenSSL from $url"
            exit 0
          fi
        done
        echo "Failed to download OpenSSL source from all mirror candidate URLs" >&2
        exit 1
      args:
        executable: /bin/bash

    - name: Extract OpenSSL source
      unarchive:
        src: "{{ "{{" }} source_root {{ "}}" }}/openssl-{{ "{{" }} openssl_version {{ "}}" }}.tar.gz"
        dest: "{{ "{{" }} source_root {{ "}}" }}"
        remote_src: yes

    - name: Download Nginx source
      shell: |
        DEST="{{ "{{" }} source_root {{ "}}" }}/nginx-{{ "{{" }} nginx_version {{ "}}" }}.tar.gz"
        if [ -f "$DEST" ] && [ $(stat -c%s "$DEST" 2>/dev/null || stat -f%z "$DEST" 2>/dev/null || echo 0) -gt 1000000 ]; then
          echo "Nginx source already downloaded ($DEST)."
          exit 0
        fi
        for url in "https://nginx.org/download/nginx-{{ "{{" }} nginx_version {{ "}}" }}.tar.gz" "https://mirrors.sohu.com/nginx/nginx-{{ "{{" }} nginx_version {{ "}}" }}.tar.gz"; do
          echo "Downloading Nginx from mirror: $url"
          if curl -f -L -C - --retry 5 --retry-delay 2 --connect-timeout 15 -sS -o "$DEST" "$url" || wget -c --tries=5 --timeout=30 -q -O "$DEST" "$url"; then
            echo "Successfully downloaded Nginx from $url"
            exit 0
          fi
        done
        echo "Failed to download Nginx source from all mirrors" >&2
        exit 1
      args:
        executable: /bin/bash

    - name: Extract Nginx source
      unarchive:
        src: "{{ "{{" }} source_root {{ "}}" }}/nginx-{{ "{{" }} nginx_version {{ "}}" }}.tar.gz"
        dest: "{{ "{{" }} source_root {{ "}}" }}"
        remote_src: yes

    - name: Configure Nginx build
      shell: >
        ./configure
        --prefix={{ "{{" }} install_path {{ "}}" }}
        --user={{ "{{" }} run_user {{ "}}" }}
        --group={{ "{{" }} run_group {{ "}}" }}
        --with-http_ssl_module
        --with-http_v2_module
        --with-http_stub_status_module
        --with-http_gzip_static_module
        --with-http_sub_module
        --with-pcre={{ "{{" }} source_root {{ "}}" }}/pcre-{{ "{{" }} pcre_version {{ "}}" }}
        --with-openssl={{ "{{" }} source_root {{ "}}" }}/openssl-{{ "{{" }} openssl_version {{ "}}" }}
      args:
        chdir: "{{ "{{" }} source_root {{ "}}" }}/nginx-{{ "{{" }} nginx_version {{ "}}" }}"

    - name: Compile and install Nginx
      shell: make -j$(nproc) && make install
      args:
        chdir: "{{ "{{" }} source_root {{ "}}" }}/nginx-{{ "{{" }} nginx_version {{ "}}" }}"

    - name: Create vhost and conf directories
      file:
        path: "{{ "{{" }} item {{ "}}" }}"
        state: directory
        mode: '0755'
      loop:
        - "{{ "{{" }} install_path {{ "}}" }}/conf/vhost"
        - "{{ "{{" }} install_path {{ "}}" }}/logs"

    - name: Write nginx.conf
      copy:
        dest: "{{ "{{" }} install_path {{ "}}" }}/conf/nginx.conf"
        content: |
          {{ .NginxBaseConfig | indent 10 }}

    - name: Write proxy.conf
      copy:
        dest: "{{ "{{" }} install_path {{ "}}" }}/conf/proxy.conf"
        content: |
          {{ .NginxProxyConfig | indent 10 }}

    - name: Write systemd unit file
      copy:
        dest: "{{ "{{" }} systemd_unit_path {{ "}}" }}"
        content: |
          {{ .NginxSystemdUnit | indent 10 }}

    - name: Write logrotate config
      copy:
        dest: /etc/logrotate.d/nginx
        content: |
          {{ .NginxLogrotate | indent 10 }}

    - name: Reload systemd daemon
      systemd:
        daemon_reload: yes

    - name: Enable and start Nginx service
      systemd:
        name: nginx
        state: started
        enabled: yes
`,
}

// PlaybookVars represents variables to inject into playbooks.
type PlaybookVars struct {
	TargetGroup       string
	DataRoot          string
	NetworkName       string
	CIDR              string
	NamePrefix        string
	MySQLImage        string
	MySQLPort         int
	MySQLRootPassword string
	RedisImage        string
	RedisPort         int
	RedisPassword     string
	RabbitMQImage     string
	RabbitMQPort      int
	RabbitMQUIPort    int
	RabbitMQUser      string
	RabbitMQPwd       string
	// Nginx binary driver fields
	NginxVersion         string
	NginxPCREVersion     string
	NginxOpenSSLVersion  string
	NginxOpenSSLURL      string
	NginxOpenSSLURLs     []string
	NginxInstallPath     string
	NginxSourceRoot      string
	NginxWWWRoot         string
	NginxSSLRoot         string
	NginxWWWLogsRoot     string
	NginxRunUser         string
	NginxRunGroup        string
	NginxSystemdUnitPath string
	// Pre-rendered nginx config file contents (auto-populated by GeneratePlaybookFile).
	// These ensure the Ansible and binary driver write identical configuration.
	NginxBaseConfig  string
	NginxProxyConfig string
	NginxSystemdUnit string
	NginxLogrotate   string
	// Pre-rendered Docker service configs (auto-populated by GeneratePlaybookFile).
	MySQLMyCnf   string
	RedisCnf     string
	RabbitMQConf string
}

// indentLines prefixes every line of s (except the first) with spaces*indent.
func indentLines(spaces int, s string) string {
	pad := strings.Repeat(" ", spaces)
	lines := strings.Split(s, "\n")
	for i := 1; i < len(lines); i++ {
		if lines[i] != "" {
			lines[i] = pad + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

// GeneratePlaybookFile parses the playbook template and writes it to a temporary file.
// For the nginx service the pre-rendered config file contents are automatically
// populated from pkg/nginxconf so both the local binary driver and the Ansible
// path write byte-for-byte identical configuration.
func GeneratePlaybookFile(tempDir string, serviceName string, vars PlaybookVars) (string, error) {
	if vars.TargetGroup == "" {
		vars.TargetGroup = "all"
	}
	// Auto-populate pre-rendered nginx config contents from the shared package.
	switch serviceName {
	case "nginx":
		if len(vars.NginxOpenSSLURLs) == 0 && vars.NginxOpenSSLVersion != "" {
			vars.NginxOpenSSLURLs = versionutil.GetOpenSSLDownloadURLs(vars.NginxOpenSSLVersion)
		}
		if vars.NginxOpenSSLURL == "" && len(vars.NginxOpenSSLURLs) > 0 {
			vars.NginxOpenSSLURL = vars.NginxOpenSSLURLs[0]
		}
		cfg := nginxconf.Config{
			InstallPath:     vars.NginxInstallPath,
			WWWRoot:         vars.NginxWWWRoot,
			SSLRoot:         vars.NginxSSLRoot,
			WWWLogsRoot:     vars.NginxWWWLogsRoot,
			RunUser:         vars.NginxRunUser,
			RunGroup:        vars.NginxRunGroup,
			SystemdUnitPath: vars.NginxSystemdUnitPath,
		}
		vars.NginxBaseConfig = nginxconf.RenderBaseConfig(cfg)
		vars.NginxProxyConfig = nginxconf.RenderProxyConfig()
		vars.NginxSystemdUnit = nginxconf.RenderSystemdUnit(cfg)
		vars.NginxLogrotate = nginxconf.RenderLogrotate(cfg)
	case "mysql":
		vars.MySQLMyCnf = mysqlconf.RenderMyCnf()
	case "redis":
		vars.RedisCnf = redisconf.RenderRedisCnf(vars.RedisPassword)
	case "rabbitmq":
		vars.RabbitMQConf = rabbitmqconf.RenderRabbitMQConf(vars.RabbitMQUser, vars.RabbitMQPwd)
	}

	tmplStr, exists := PlaybookTemplates[serviceName]
	if !exists {
		return "", fmt.Errorf("playbook template for service %s not found", serviceName)
	}

	// Register custom template functions.
	funcMap := template.FuncMap{
		"indent": indentLines,
		"join":   strings.Join,
	}

	tmpl, err := template.New(serviceName).Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse playbook template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to execute playbook template: %w", err)
	}

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp dir %s: %w", tempDir, err)
	}

	tempFile := filepath.Join(tempDir, fmt.Sprintf("%s_playbook.yml", serviceName))
	if err := os.WriteFile(tempFile, buf.Bytes(), 0600); err != nil {
		return "", fmt.Errorf("failed to write playbook file: %w", err)
	}

	return tempFile, nil
}
