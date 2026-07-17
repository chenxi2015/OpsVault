package ansible

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

    - name: Set default Docker Repo Version
      set_fact:
        docker_repo_ver: "{{ "{{" }} ansible_distribution_major_version {{ "}}" }}"
      when: docker_check.rc != 0

    - name: Ensure fallback Docker Repo Version is 7
      set_fact:
        docker_repo_ver: "7"
      when:
        - docker_check.rc != 0
        - docker_repo_ver not in ['7', '8', '9']

    - name: Override Docker Repo Version for TencentOS 3
      set_fact:
        docker_repo_ver: "8"
      when:
        - docker_check.rc != 0
        - ansible_distribution | lower == 'tencentos'
        - ansible_distribution_major_version | int == 3

    - name: Override Docker Repo Version for TencentOS 4
      set_fact:
        docker_repo_ver: "9"
      when:
        - docker_check.rc != 0
        - ansible_distribution | lower == 'tencentos'
        - ansible_distribution_major_version | int == 4

    - name: Add Docker CE repository
      copy:
        dest: /etc/yum.repos.d/docker-ce.repo
        content: |
          [docker-ce-stable]
          name=Docker CE Stable
          baseurl=https://mirrors.cloud.tencent.com/docker-ce/linux/centos/{{ "{{" }} docker_repo_ver {{ "}}" }}/$basearch/stable
          enabled=1
          gpgcheck=1
          gpgkey=https://mirrors.cloud.tencent.com/docker-ce/linux/centos/gpg
        mode: '0644'
      when: docker_check.rc != 0

    - name: Install Docker CE
      yum:
        name:
          - docker-ce
          - docker-ce-cli
          - containerd.io
        state: present
      when: docker_check.rc != 0

{{- if .RegistryMirrors }}
    - name: Ensure /etc/docker directory exists
      file:
        path: /etc/docker
        state: directory
        mode: '0755'

    - name: Configure Docker registry mirrors
      copy:
        dest: /etc/docker/daemon.json
        content: |
          {
            "registry-mirrors": [
              {{- range $i, $m := .RegistryMirrors }}
              {{- if $i }},{{ end }}
              "{{ $m }}"
              {{- end }}
            ]
          }
      register: docker_mirror_config

    - name: Restart Docker if daemon.json changed
      systemd:
        name: docker
        state: restarted
        daemon_reload: yes
      when: docker_mirror_config.changed
{{- end }}

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

{{- if .RegistryMirrors }}
    - name: Ensure /etc/docker directory exists
      file:
        path: /etc/docker
        state: directory
        mode: '0755'

    - name: Configure Docker registry mirrors
      copy:
        dest: /etc/docker/daemon.json
        content: |
          {
            "registry-mirrors": [
              {{- range $i, $m := .RegistryMirrors }}
              {{- if $i }},{{ end }}
              "{{ $m }}"
              {{- end }}
            ]
          }
      register: docker_mirror_config

    - name: Restart Docker if daemon.json changed
      systemd:
        name: docker
        state: restarted
        daemon_reload: yes
      when: docker_mirror_config.changed
{{- end }}

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
        -e MYSQL_ROOT_PASSWORD='{{ .MySQLRootPassword }}'
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

{{- if .RegistryMirrors }}
    - name: Ensure /etc/docker directory exists
      file:
        path: /etc/docker
        state: directory
        mode: '0755'

    - name: Configure Docker registry mirrors
      copy:
        dest: /etc/docker/daemon.json
        content: |
          {
            "registry-mirrors": [
              {{- range $i, $m := .RegistryMirrors }}
              {{- if $i }},{{ end }}
              "{{ $m }}"
              {{- end }}
            ]
          }
      register: docker_mirror_config

    - name: Restart Docker if daemon.json changed
      systemd:
        name: docker
        state: restarted
        daemon_reload: yes
      when: docker_mirror_config.changed
{{- end }}

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

{{- if .RegistryMirrors }}
    - name: Ensure /etc/docker directory exists
      file:
        path: /etc/docker
        state: directory
        mode: '0755'

    - name: Configure Docker registry mirrors
      copy:
        dest: /etc/docker/daemon.json
        content: |
          {
            "registry-mirrors": [
              {{- range $i, $m := .RegistryMirrors }}
              {{- if $i }},{{ end }}
              "{{ $m }}"
              {{- end }}
            ]
          }
      register: docker_mirror_config

    - name: Restart Docker if daemon.json changed
      systemd:
        name: docker
        state: restarted
        daemon_reload: yes
      when: docker_mirror_config.changed
{{- end }}

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
        -e RABBITMQ_DEFAULT_USER='{{ .RabbitMQUser }}'
        -e RABBITMQ_DEFAULT_PASS='{{ .RabbitMQPwd }}'
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

	"push": `---
- name: Push OpsVault binary and config to target nodes
  hosts: {{ .TargetGroup }}
  become: yes
  tasks:
    - name: Ensure target bin and configs directories exist
      file:
        path: "{{ "{{" }} item {{ "}}" }}"
        state: directory
        mode: '0755'
      loop:
        - "{{ .DataRoot }}/bin"
        - "{{ .DataRoot }}/configs"

    - name: Copy OpsVault executable binary
      copy:
        src: "{{ .BinaryPath }}"
        dest: "{{ .DataRoot }}/bin/opsvault"
        mode: '0755'

    - name: Copy default configuration file
      copy:
        src: "{{ .ConfigPath }}"
        dest: "{{ .DataRoot }}/configs/default.yaml"
        mode: '0644'
        force: {{ if .Force }}yes{{ else }}no{{ end }}
        backup: yes

    - name: Create global symlink for opsvault in /usr/local/bin
      file:
        src: "{{ .DataRoot }}/bin/opsvault"
        dest: /usr/local/bin/opsvault
        state: link
        force: yes
`,

	"minio": `---
- name: Deploy MinIO via Docker
  hosts: {{ .TargetGroup }}
  become: yes
  tasks:
    - name: Create MinIO data directory
      file:
        path: "{{ .DataRoot }}/minio/data"
        state: directory
        mode: '0755'

{{- if .RegistryMirrors }}
    - name: Ensure /etc/docker directory exists
      file:
        path: /etc/docker
        state: directory
        mode: '0755'

    - name: Configure Docker registry mirrors
      copy:
        dest: /etc/docker/daemon.json
        content: |
          {
            "registry-mirrors": [
              {{- range $i, $m := .RegistryMirrors }}
              {{- if $i }},{{ end }}
              "{{ $m }}"
              {{- end }}
            ]
          }
      register: docker_mirror_config

    - name: Restart Docker if daemon.json changed
      systemd:
        name: docker
        state: restarted
        daemon_reload: yes
      when: docker_mirror_config.changed
{{- end }}

    - name: Create Docker bridge network if not exists
      shell: "docker network inspect {{ .NetworkName }} || docker network create --subnet={{ .CIDR }} {{ .NetworkName }}"
      register: network_create
      changed_when: "'Created' in network_create.stdout"

    - name: Stop and remove existing MinIO container
      shell: "docker rm -f {{ .NamePrefix }}-minio || true"

    - name: Run MinIO container
      shell: >
        docker run -d
        --name {{ .NamePrefix }}-minio
        --restart always
        --network {{ .NetworkName }}
        -p {{ .MinIOPort }}:9000
        -p {{ .MinIOConsolePort }}:9001
        -v {{ .DataRoot }}/minio/data:/data
        -e MINIO_ROOT_USER='{{ .MinIORootUser }}'
        -e MINIO_ROOT_PASSWORD='{{ .MinIORootPassword }}'
        {{ .MinIOImage }}
        server /data --console-address :9001
`,
}

// UninstallTemplates contains built-in ansible playbooks for uninstallation and purging.
var UninstallTemplates = map[string]string{
	"mysql": `---
- name: Uninstall MySQL Docker Service
  hosts: {{ .TargetGroup }}
  become: yes
  tasks:
    - name: Stop and remove MySQL container
      shell: "docker rm -f {{ .NamePrefix }}-mysql || true"

{{- if .Purge }}
    - name: Purge MySQL data directory
      file:
        path: "{{ .DataRoot }}/mysql"
        state: absent
{{- end }}
`,

	"redis": `---
- name: Uninstall Redis Docker Service
  hosts: {{ .TargetGroup }}
  become: yes
  tasks:
    - name: Stop and remove Redis container
      shell: "docker rm -f {{ .NamePrefix }}-redis || true"

{{- if .Purge }}
    - name: Purge Redis data directory
      file:
        path: "{{ .DataRoot }}/redis"
        state: absent
{{- end }}
`,

	"rabbitmq": `---
- name: Uninstall RabbitMQ Docker Service
  hosts: {{ .TargetGroup }}
  become: yes
  tasks:
    - name: Stop and remove RabbitMQ container
      shell: "docker rm -f {{ .NamePrefix }}-rabbitmq || true"

{{- if .Purge }}
    - name: Purge RabbitMQ data directory
      file:
        path: "{{ .DataRoot }}/rabbitmq"
        state: absent
{{- end }}
`,

	"nginx": `---
- name: Uninstall Nginx Binary Service
  hosts: {{ .TargetGroup }}
  become: yes
  tasks:
    - name: Stop Nginx systemd service
      shell: "systemctl stop nginx || true"

    - name: Disable and remove Nginx systemd service
      shell: |
        systemctl disable nginx || true
        rm -f /lib/systemd/system/nginx.service
        systemctl daemon-reload

    - name: Remove Nginx installation directory
      file:
        path: "{{ .NginxInstallPath }}"
        state: absent

{{- if .Purge }}
    - name: Purge Nginx website root directory
      file:
        path: "{{ .NginxWWWRoot }}"
        state: absent

    - name: Purge Nginx SSL certificate directory
      file:
        path: "{{ .NginxSSLRoot }}"
        state: absent

    - name: Purge Nginx logs directory
      file:
        path: "{{ .NginxWWWLogsRoot }}"
        state: absent
{{- end }}
`,

	"docker": `---
- name: Uninstall Docker Engine
  hosts: {{ .TargetGroup }}
  become: yes
  tasks:
    - name: Stop and disable Docker service
      systemd:
        name: docker
        state: stopped
        enabled: no
      ignore_errors: yes

    - name: Remove Docker CE packages
      yum:
        name:
          - docker-ce
          - docker-ce-cli
          - containerd.io
        state: absent

{{- if .Purge }}
    - name: Purge Docker system root and opsvault network
      shell: |
        rm -rf /var/lib/docker
        rm -rf /etc/docker
{{- end }}
`,

	"minio": `---
- name: Uninstall MinIO Docker Service
  hosts: {{ .TargetGroup }}
  become: yes
  tasks:
    - name: Stop and remove MinIO container
      shell: "docker rm -f {{ .NamePrefix }}-minio || true"

{{- if .Purge }}
    - name: Purge MinIO data directory
      file:
        path: "{{ .DataRoot }}/minio"
        state: absent
{{- end }}
`,
}

// ReloadTemplates contains built-in ansible playbooks for reloading services.
var ReloadTemplates = map[string]string{
	"nginx": `---
- name: Reload Nginx Service
  hosts: {{ .TargetGroup }}
  become: yes
  tasks:
    - name: Reload Nginx systemd service
      systemd:
        name: nginx
        state: reloaded
`,
}
