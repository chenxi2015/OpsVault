---
title: OpsVault 运维工具箱完整开发需求文档
version: 1.0
date: 2026-07-02
tags: golang,cobra,bubbletea,运维工具,docker,nginx,中间件部署
target_os: CentOS7 / CentOS Stream
---

# 1 项目总览

## 1.1 项目名称
OpsVault（运维百宝箱）

## 1.2 核心定位
Go语言开发一站式CentOS中间件&Web服务运维CLI+TUI工具集；**默认Docker部署驱动，预留二进制驱动扩展**；Nginx单独采用 OpsVault 内建 BinaryDriver 源码编译裸机部署，其余组件优先Docker桥接网络部署。

## 1.3 强制技术栈约束
*   **CLI基座（固定使用）**：`spf13/cobra` + `spf13/viper` 
    * 创建cli使用cobra-cli 相关命令来创建 如：`cobra-cli add serve`
    *   提供分层子命令、全局配置、shell自动补全、全局持久flag。
*   **TUI交互基座（固定使用）**：`charmbracelet/bubbletea` + `bubbles` + `lipgloss`
    *   实现终端可视化面板、交互式表格、配置向导、流式日志、Markdown彩色渲染。
*   **容器操作SDK**：`github.com/docker/docker/client`
    *   统一管理docker网桥、容器生命周期、镜像拉取、宿主机数据挂载。
*   **系统底层工具**：Go标准库 `os/exec` / `os` / `net` 等
    *   封装PID查询、端口检测、systemd管理、文件权限、ssl证书处理。
*   **Nginx 源码安装参考**：
    *   Nginx 安装/升级/卸载必须由 OpsVault Go 代码在 `internal/driver/binary` 内实现。

## 1.4 核心架构强制约束
### 部署驱动抽象层（固定接口，不可修改）
文件路径：[driver.go](./internal/driver/driver.go)
所有中间件必须实现该通用接口：
```go
type ServiceDriver interface {
	Install() error
	Start() error
	Stop() error
	Restart() error
	Uninstall(purgeData bool) error
	Upgrade(targetVersion string) error
	Status() (*ServiceStatus, error)
}
```
*   **DockerDriver**：实现 MySQL / Redis / RocketMQ / RabbitMQ / PostgreSQL 驱动。
*   **BinaryDriver**：当前仅实现 Nginx；后续全组件可扩展二进制部署命令。
*   Cobra 命令统一通过 `--mode docker|binary` 切换驱动，默认 `docker` 模式。

### Docker 全局统一规范（不可修改）
*   **专属网桥名称**：`opsvault-net`，默认 CIDR `172.28.0.0/16`。
*   工具启动自动检测网桥，不存在则自动创建。
*   所有容器统一加入该网桥，容器间互通访问。
*   **宿主机持久化根目录固定**：`/data/opsvault/`，各组件独立子目录存储数据 & 日志。
*   **容器命名统一前缀**：`opsvault-{组件名}`。
*   卸载命令统一参数 `--purge`，不带参数保留宿主机数据目录，携带则彻底删除数据。

### Nginx 专属强制规则
*   禁止 Docker 部署 Nginx，仅 Binary 驱动。
*   安装逻辑由 OpsVault 内建源码安装器完成：安装编译依赖、下载 Nginx/PCRE/OpenSSL 源码、生成配置并注册 systemd。
*   **Nginx 目录规范**：
    *   程序路径：`/usr/local/nginx`
    *   网站根目录：`/data/wwwroot`
    *   SSL 证书目录：`/data/ssl`
*   **进程托管**：systemd 管理 nginx 服务。

---

# 2 完整项目目录结构（代码生成严格遵循）
```plaintext
OpsVault/
├── cmd/                    # Cobra命令入口
│   ├── root.go             # 根命令、viper初始化、docker客户端全局加载、全局flag
│   ├── tui.go              # 统一TUI总入口：opsvault tui
│   ├── nginx/              # Nginx二进制子命令组
│   │   ├── install.go
│   │   ├── start.go
│   │   ├── stop.go
│   │   ├── restart.go
│   │   ├── uninstall.go
│   │   ├── upgrade.go
│   │   ├── vhost.go        # 虚拟主机增删查改
│   │   ├── ssl.go          # SSL证书申请/绑定/续期/删除
│   │   └── status.go
│   ├── mysql/              # Docker部署MySQL
│   ├── redis/              # Docker部署Redis
│   ├── rocketmq/           # Docker部署RocketMQ
│   ├── rabbitmq/           # Docker部署RabbitMQ
│   ├── postgres/           # 预留PostgreSQL（Docker驱动）
│   └── ansible/            # Ansible多机编排、下发分发与回收
│       ├── root.go         # 根入口与多环境加载 (-e test|prod)
│       ├── ping.go         # 批量连通性测试
│       ├── exec.go         # ad-hoc shell并发执行
│       ├── doctor.go       # 多机负载/系统与中间件体检
│       ├── list.go         # 主机分组与资产清单概览
│       ├── deploy.go       # 批量一键中间件编排部署 (生成强密码并持久化)
│       ├── push.go         # 边缘推流：下发二进制和配置文件至被控端
│       └── uninstall.go    # 远程集群批量回收与深度清理 (--purge)
├── internal/               # 核心业务逻辑、驱动抽象
│   ├── driver/
│   │   ├── driver.go       # ServiceDriver统一接口定义
│   │   ├── ansible/        # Ansible驱动编排包
│   │   │   ├── executor.go # Ansible Executor封装与并发调起
│   │   │   └── playbook.go # PlaybookTemplates、UninstallTemplates及动态Inventory渲染
│   │   ├── docker/         # Docker驱动实现包
│   │   │   ├── network.go  # opsvault-net网桥创建/校验工具
│   │   │   ├── base.go
│   │   │   ├── mysql.go
│   │   │   ├── redis.go
│   │   │   ├── rocketmq.go
│   │   │   ├── rabbitmq.go
│   │   │   └── postgres.go
│   │   └── binary/         # 二进制驱动扩展包
│   │       ├── base.go
│   │       ├── nginx.go    # Nginx BinaryDriver生命周期、vhost/SSL管理
│   │       └── nginx_installer.go # Nginx源码安装、配置生成、systemd/logrotate注册
│   ├── system/             # 宿主机系统工具包
│   │   ├── port.go         # 端口占用检测
│   │   ├── proc.go         # PID进程查询、进程详情
│   │   ├── sysctl.go       # 内核文件句柄、网络参数优化
│   │   └── systemd.go      # 二进制服务注册/启停管理
│   └── tui/                # BubbleTea全TUI页面
│       ├── root_model.go   # TUI根模型
│       ├── dashboard.go    # 全局服务总览仪表盘
│       ├── nginx_panel.go  # Nginx可视化vhost/ssl管理
│       ├── docker_panel.go # 容器统一管理面板
│       └── config_wizard.go# 首次运行配置向导（网桥/镜像版本/存储目录）
├── pkg/                    # 公共工具包，全局复用
│   ├── dockercli/          # docker客户端封装、网桥统一操作
│   ├── logger/             # 统一日志打印
│   ├── sslutil/            # Let’s Encrypt证书生成、绑定工具
│   ├── fileutil/           # 文件/目录权限操作
│   └── netutil/            # 网络、CIDR校验工具
├── configs/
│   └── default.yaml        # 全局默认配置模板
├── go.mod
├── go.sum
└── README.md
```

---

# 3 全局命令行规范（Cobra 分层，代码实现严格对齐）

## 3.1 全局 TUI 可视化入口
```bash
opsvault tui
```

## 3.2 Nginx（仅 binary 模式，无 docker）
```bash
# 基础生命周期
opsvault nginx install
opsvault nginx start
opsvault nginx stop
opsvault nginx restart
opsvault nginx uninstall [--purge]
opsvault nginx upgrade

# 虚拟主机vhost管理
opsvault nginx vhost add --domain shturl. --root /data/wwwroot/xxx
opsvault nginx vhost del --domain shturl. [--delete-root]
opsvault nginx vhost list

# SSL证书管理
opsvault nginx ssl apply --domain shturl.
opsvault nginx ssl renew --domain shturl.
opsvault nginx ssl delete --domain shturl.

# 状态与日志
opsvault nginx status
opsvault nginx log
```

## 3.3 MySQL（Docker 驱动，预留 --mode binary 后续扩展）
```bash
opsvault mysql install [--root-pwd xxx]
opsvault mysql start
opsvault mysql stop
opsvault mysql restart
opsvault mysql uninstall [--purge]
opsvault mysql upgrade --tag 8.4
opsvault mysql status
opsvault mysql log
```

## 3.4 Redis（Docker 驱动）
```bash
opsvault redis install [--pwd xxx]
opsvault redis start
opsvault redis stop
opsvault redis restart
opsvault redis uninstall [--purge]
opsvault redis upgrade --tag 7.2-alpine
opsvault redis status
```

## 3.5 RocketMQ（Docker 驱动）
```bash
opsvault rocketmq install
opsvault rocketmq start
opsvault rocketmq stop
opsvault rocketmq restart
opsvault rocketmq uninstall [--purge]
opsvault rocketmq upgrade --tag 5.3.0
opsvault rocketmq version  # 查询Broker版本
opsvault rocketmq dlq stat # 死信队列统计
opsvault rocketmq log
```

## 3.6 RabbitMQ（Docker 驱动）
```bash
opsvault rabbitmq install [--admin-user admin --admin-pwd 123456]
opsvault rabbitmq start
opsvault rabbitmq stop
opsvault rabbitmq restart
opsvault rabbitmq uninstall [--purge]
opsvault rabbitmq upgrade --tag 3.13-management
opsvault rabbitmq status
```

## 3.7 PostgreSQL（预留，Docker 驱动）
```bash
opsvault postgres install
opsvault postgres start
opsvault postgres stop
opsvault postgres uninstall
opsvault postgres upgrade
```

## 3.8 Ansible 批量集群编排与边缘协作 (`cmd/ansible`)
```bash
# 基础连接、连通性与执行
opsvault ansible ping --group db_servers [-e test|prod]
opsvault ansible exec --cmd "uptime" --group db_servers [-e test|prod]
opsvault ansible doctor --group db_servers [-e test|prod]
opsvault ansible list [-e test|prod]

# 批量一键中介件编排部署与回收
opsvault ansible deploy --service mysql --group db_servers
opsvault ansible uninstall --service mysql --group db_servers [--purge]

# 边缘推流初始化 (推送可执行文件与配置至 /data/opsvault/ 并创建全局软连)
opsvault ansible push --group db_servers --bin ./bin/opsvault-linux-amd64 --config-path ./configs/default.yaml
```

---

# 4 各组件功能完整需求

## 4.1 Nginx 模块（OpsVault BinaryDriver 源码编译）
### 4.1.1 Install
*   自动检测 CentOS 系统依赖，yum 安装编译依赖。
*   下载 Nginx、PCRE、OpenSSL 源码至配置的 `nginx.source_root`。
*   按 Nginx 编译参数进行 Go 内建编排，不调用 shell 入口脚本。
*   自动生成 `nginx.conf`、`proxy.conf`、systemd unit、logrotate 配置。
*   自动注册 systemd 服务、开启开机自启、优化 ulimit 文件句柄。

### 4.1.2 vhost 管理
*   **add**：自动生成 nginx conf、80 端口站点配置、自动创建网站根目录。
*   **del**：删除站点 conf，可选同步删除网站目录。
*   **list**：输出表格展示域名、根目录、SSL 启用状态、监听端口。

### 4.1.3 SSL 证书
*   **apply**：自动申请 Let’s Encrypt 免费证书，修改 vhost 配置强制跳转 HTTPS。
*   **renew**：批量续期全部域名证书。
*   **delete**：删除证书文件，恢复 80 纯 HTTP 配置。

### 4.1.4 启停 / 卸载 / 升级
*   **start/stop/restart**：调用 `systemctl nginx`。
*   **upgrade**：更新 `nginx.version` 后复用 OpsVault 内建源码安装器重新编译安装并重启 Nginx。
*   **uninstall**：停止并禁用 systemd 服务，删除 Nginx 程序目录、systemd unit、logrotate 配置，可选删除 `/data/wwwroot`、`/data/ssl`、`/data/wwwlogs`。

## 4.2 MySQL（Docker 网桥部署）
*   **install**：拉取配置内镜像 tag，创建容器，挂载 `/data/opsvault/mysql`，加入 `opsvault-net` 网桥，映射 3306 端口，自定义 root 密码。
*   **start/stop/restart**：容器生命周期管理。
*   **uninstall**：`--purge` 删除宿主机数据目录，否则保留。
*   **upgrade**：停止旧容器，拉取新镜像，复用原有数据目录重建容器。
*   **status**：输出容器运行状态、存储占用、端口映射、网桥名称。

## 4.3 Redis（Docker 网桥部署）
*   **install**：开启 RDB 持久化，自定义访问密码，网桥接入，映射 6379 端口。
*   启停、卸载、版本升级逻辑同 MySQL。

## 4.4 RocketMQ（Docker 一体化 NameServer+Broker）
*   容器内置 NameServer+Broker，统一接入 `opsvault-net` 网桥。
*   映射 9876 端口，CommitLog 持久化至宿主机 `/data/opsvault/rocketmq`。
*   **专属子命令**：`version` 查询 Broker 版本、`dlq stat` 统计死信堆积数量。
*   支持启停、卸载、镜像版本升级。

## 4.5 RabbitMQ（Docker）
*   管理面板 15672 端口映射，自定义管理员账号密码。
*   容器加入 `opsvault-net`，可与 RocketMQ/MySQL 容器互通访问。
*   完整生命周期命令，升级更换镜像 tag 不丢失数据。

## 4.6 PostgreSQL（预留迭代）
*   逻辑完全对齐 MySQL Docker 驱动，端口 5432，独立宿主机数据目录，后续可扩展 BinaryDriver 二进制安装。

## 4.7 Ansible 批量驱动与边缘协同模块 (`internal/driver/ansible`)
*   **多机连通性与执行**：支持并发 SSH Ping 与 ad-hoc 执行，自动从 CLI 提取当前激活的 Cobra 参数并与 Viper 配置合并。
*   **多机健康诊断 (doctor)**：获取并解析集群多主机 CPU、内存、根分区磁盘及 `docker`/`nginx` 服务运行状态，通过 `lipgloss` 渲染终端表格。
*   **安全凭据生成与下发**：当 `ansible deploy` 部署 MySQL / Redis / RabbitMQ 时若未设定凭据，系统由 `credutil.GenPassword(20)` 自动生成 20 位高强度强密码，嵌入 Playbook 批量渲染下发，并在终端展示高亮卡片。
*   **边缘协同推送 (push)**：推送到目标节点 `/data/opsvault/bin` 和 `/configs` 中并生成 `/usr/local/bin/opsvault` 全局链接，赋予每个节点本地就地运行 `opsvault tui` 的独立运维能力。
*   **批量远程优雅回收 (uninstall)**：根据 `serviceName` 批量调用 Docker/systemd 停止并卸载组件，配合 `--purge` 开关控制 `/data/opsvault/` 挂载目录物理卸载。

---

# 5 配置文件 configs/default.yaml 标准模板
```yaml
# Docker全局网桥配置
docker:
  network_name: "opsvault-net"
  cidr: "172.28.0.0/16"
  data_root: "/data/opsvault"
  # 默认镜像版本
  images:
    mysql: "mysql:8.0"
    redis: "redis:7-alpine"
    rocketmq: "apache/rocketmq:5.3.0"
    rabbitmq: "rabbitmq:3-management"
    postgres: "postgres:15"
  # 容器资源限制
  resource_limit:
    cpu_max: "2"
    mem_max: "2g"

# Nginx BinaryDriver源码编译配置
nginx:
  install_path: "/usr/local/nginx"
  www_root: "/data/wwwroot"
  ssl_root: "/data/ssl"
  wwwlogs_root: "/data/wwwlogs"
  source_root: "/usr/local/src/opsvault-nginx"
  version: "1.31.0"
  pcre_version: "8.45"
  openssl_version: "1.1.1w"
  run_user: "www"
  run_group: "www"
  systemd_unit_path: "/lib/systemd/system/nginx.service"
  logrotate_path: "/etc/logrotate.d/nginx"

# 全局日志配置
log:
  level: info
  storage_path: "/data/opsvault/logs"
```

---

# 6 TUI 交互功能需求（BubbleTea 完整实现）
*   **全局仪表盘**：表格展示 Nginx 进程状态、所有 Docker 容器运行状态、端口、磁盘占用。
*   **Nginx 可视化面板**：可视化增删 vhost、一键申请 SSL、实时滚动日志、证书管理。
*   **Docker 容器管理面板**：可视化启停 / 升级 / 卸载容器，分页查看容器日志。
*   **初始化配置向导**：首次运行交互式配置网桥网段、宿主机存储根目录、默认镜像 tag。
*   **终端增强**：彩色区分日志级别、Markdown 代码块渲染、加载动画、分页表格、快捷键操作。

---

# 7 扩展迭代规划（预留扩展入口，不破坏现有架构）
*   **一期完成**：Nginx (OpsVault BinaryDriver源码编译) + MySQL/Redis/RocketMQ/RabbitMQ (Docker)
*   **二期开发**：PostgreSQL Docker 完整功能
*   **三期扩展**：为 MySQL/Redis/RocketMQ 新增 BinaryDriver 二进制部署模式
*   **四期扩展**：集成 LLM AI 对话 TUI 子命令，自动分析中间件报错、死信堆积故障
*   **五期完成**：多服务器 Ansible 批量集群巡检 (doctor/list/ping/exec)、中间件批量一键部署 (deploy)、边缘节点推流初始化 (push) 与服务清理回收 (uninstall)

---

# 8 CentOS 运行前置依赖校验
*   **Docker 环境**：工具内置一键安装 docker-ce 脚本，未安装时自动执行。
*   **Nginx 二进制依赖**：OpsVault 自动安装 yum/dnf 编译依赖，无需人工干预。
*   **权限要求**：工具运行必须具备 root/sudo 权限（操作 systemd、80/443 端口、docker）。
*   **开放端口**：80、443、3306、6379、9876、15672。

---

# 9 AI 编码强制约束（必须严格遵守）
1.  所有中间件部署逻辑统一封装至 `driver` 层，Cobra 命令仅做参数接收、调用驱动、输出结果。
2.  Docker 网桥操作统一封装至 `pkg/dockercli`，禁止重复编写创建网桥代码。
3.  TUI 层与业务驱动完全解耦，TUI 复用 `driver` 层接口，不重复实现部署逻辑。
4.  禁止硬编码路径、镜像版本、网桥名称，全部读取 viper 配置文件。
5.  代码分层清晰，禁止跨包循环导入，公共工具统一放入 `pkg` 目录。
6.  所有操作增加错误捕获、日志输出，全局支持 `--debug` flag 打印详细执行日志。
7.  严格区分 Docker 驱动与 Binary 驱动，Nginx 仅使用 Binary 驱动，其余默认 Docker。
