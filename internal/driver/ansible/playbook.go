package ansible

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// PlaybookTemplates contains built-in ansible playbooks.
var PlaybookTemplates = map[string]string{
	"docker": `---
- name: Install and Setup Docker
  hosts: all
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
  hosts: all
  become: yes
  tasks:
    - name: Create MySQL data directory
      file:
        path: "{{ .DataRoot }}/mysql"
        state: directory
        mode: '0755'

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
        -v {{ .DataRoot }}/mysql:/var/lib/mysql
        -e MYSQL_ROOT_PASSWORD={{ .MySQLRootPassword }}
        {{ .MySQLImage }}
`,

	"redis": `---
- name: Deploy Redis via Docker
  hosts: all
  become: yes
  tasks:
    - name: Create Redis data directory
      file:
        path: "{{ .DataRoot }}/redis"
        state: directory
        mode: '0755'

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
        -v {{ .DataRoot }}/redis:/data
        {{ .RedisImage }}
        redis-server --requirepass "{{ .RedisPassword }}" --appendonly yes
`,

	"rabbitmq": `---
- name: Deploy RabbitMQ via Docker
  hosts: all
  become: yes
  tasks:
    - name: Create RabbitMQ data directory
      file:
        path: "{{ .DataRoot }}/rabbitmq"
        state: directory
        mode: '0755'

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
        -v {{ .DataRoot }}/rabbitmq:/var/lib/rabbitmq
        -e RABBITMQ_DEFAULT_USER={{ .RabbitMQUser }}
        -e RABBITMQ_DEFAULT_PASS={{ .RabbitMQPwd }}
        {{ .RabbitMQImage }}
`,
}

// PlaybookVars represents variables to inject into playbooks.
type PlaybookVars struct {
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
}

// GeneratePlaybookFile parses the playbook template and writes it to a temporary file.
func GeneratePlaybookFile(tempDir string, serviceName string, vars PlaybookVars) (string, error) {
	tmplStr, exists := PlaybookTemplates[serviceName]
	if !exists {
		return "", fmt.Errorf("playbook template for service %s not found", serviceName)
	}

	tmpl, err := template.New(serviceName).Parse(tmplStr)
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
