# OpsVault

```text
 ██████╗ ██████╗ ███████╗██╗   ██╗ █████╗ ██╗   ██╗██╗  ████████╗
██╔═══██╗██╔══██╗██╔════╝██║   ██║██╔══██╗██║   ██║██║  ╚══██╔══╝
██║   ██║██████╔╝███████╗██║   ██║███████║██║   ██║██║     ██║   
██║   ██║██╔═══╝ ╚════██║╚██╗ ██╔╝██╔══██║██║   ██║██║     ██║   
╚██████╔╝██║     ███████║ ╚████╔╝ ██║  ██║╚██████╔╝███████╗██║   
 ╚═════╝ ╚═╝     ╚══════╝  ╚═══╝  ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝   
```

OpsVault 是一个面向 CentOS 7 / CentOS Stream 的运维工具箱，提供：

- 基于 `cobra` + `viper` 的分层 CLI
- 基于 `bubbletea` + `lipgloss` 的 TUI 入口
- 基于 Docker SDK 的中间件驱动骨架
- 内建源码编译的 Nginx 二进制驱动骨架

## 当前已完成

- 完整命令树：
  - `opsvault tui`
  - `opsvault init` (一键初始化与服务选择安装)
  - `opsvault doctor` (系统与运行环境体检诊断)
  - `opsvault nginx ...`
  - `opsvault mysql ...`
  - `opsvault redis ...`
  - `opsvault rocketmq ...`
  - `opsvault rabbitmq ...`
  - `opsvault postgres ...`
  - `opsvault minio ...` (MinIO 对象存储管理，内置 Console)
  - `opsvault bak ...` (配置备份与恢复)
  - `opsvault ansible ...` (多机批量连接、巡检、自动化部署、二进制分发与服务回收)
- 全局配置加载与默认配置模板
- `driver.ServiceDriver` 统一接口与 `ServiceStatus` 状态结构
- Docker 网络/基础驱动封装
- Nginx 源码下载、编译安装、systemd 与基础配置生成
- 基础 TUI 页面骨架
- 核心单元测试与可编译构建

## 配置文件

默认配置文件位于：

- [configs/default.yaml](./configs/default.yaml)

可通过以下方式指定自定义配置：

```bash
opsvault --config /path/to/config.yaml mysql status
```

### 自动生成随机密码说明

为了保证数据库等中间件的部署安全性：
- 如果 `default.yaml` 中相关服务的密码项被配置为空（例如 `mysql.root_password: ""`、`redis.password: ""`、`postgres.password: ""`、`rabbitmq.admin_pwd: ""` 或 `minio.root_password: ""`），且在 CLI 或 TUI 安装时未额外指定自定义密码参数，系统将**自动生成一个 20 位的强随机密码**。
- 生成的随机密码会**自动持久化写回配置文件**中。这确保了后续查看状态 (`status`)、读取凭证 (`credentials`)、重启或容器重建等操作对密码读取的一致性。

## 配置备份与恢复 (bak)

`opsvault bak` 子命令用于备份和恢复各中间件服务的配置文件（如 MySQL 的 `my.cnf`、Redis 的 `redis.conf`、Nginx 的 `conf/` 目录等）以及全局 `default.yaml` 配置文件。

### 1. 创建备份
备份文件默认以 `backup_YYYYMMDD_HHMMSS` 命名，也可以自定义名称和描述。

```bash
# 备份所有服务的配置及全局配置
opsvault bak create --desc "系统初始化配置备份"

# 备份指定服务的配置（如仅备份 mysql 和 nginx）
opsvault bak create mysql nginx --name my-init-backup
```

### 2. 查看备份列表
```bash
opsvault bak list
```

### 3. 恢复配置
恢复指定备份的配置文件，在执行恢复前有安全确认交互，也可使用 `-f` 强行覆盖。

```bash
# 恢复备份中的所有服务配置
opsvault bak restore my-init-backup

# 仅恢复备份中的 nginx 配置
opsvault bak restore my-init-backup nginx

# 强制恢复无需确认交互
opsvault bak restore my-init-backup -f
```

### 4. 删除备份
```bash
opsvault bak delete my-init-backup
```

## 构建与测试

```bash
go test ./...
go build ./...
```

## 说明

当前版本重点完成项目基础设施与命令/驱动骨架，已经满足：

- 命令分层完整
- 驱动层和命令层解耦
- Docker 统一走 `pkg/dockercli`
- Nginx 统一走 Binary Driver

仍需继续深化的部分包括：

- Docker 容器创建参数与健康检查细节
- RocketMQ 死信统计真实实现
- Nginx vhost/SSL 配置联动与 HTTPS 强制跳转
- TUI 与实时驱动状态联动
- 更细粒度测试覆盖

## Ansible 批量多机运维命令 (ansible)

`opsvault ansible` 子命令组允许您从本地控制端（如您的 Mac 笔记本）批量对多台远程服务器进行环境测试、Shell 命令执行、多机系统巡检以及中间件一键部署。

### 1. 准备工作

*   **控制机 (如 Mac 笔记本)**：需要安装 `ansible` 和 `ansible-playbook`。
    ```bash
    # 在 macOS 上使用 Homebrew 安装
    brew install ansible
    ```
*   **受控远程服务器**：只需开启 SSH 且内置 Python，**无需**安装任何 Ansible 代理或客户端。
*   **配置主机清单**：
    
    Ansible 使用独立的配置文件。首先需要将示例配置 `configs/ansible.example.yaml` 复制为 `configs/ansible.yaml`：
    
    ```bash
    cp configs/ansible.example.yaml configs/ansible.yaml
    ```
    
    然后在 `configs/ansible.yaml` 中配置远程主机组、IP、SSH端口、账号及密码/私钥：
    
    ```yaml
    ansible:
      bin_path: "ansible"
      playbook_bin_path: "ansible-playbook"
      temp_dir: "./ansible_tmp"                     # 临时清单文件生成目录
      inventory:
        groups:
          - name: "db_servers"
            hosts:
              - ip: "192.168.1.10"
                port: 22
                user: "root"
                ssh_password: "YourPassword"         # 使用密码登录时填入
                ssh_private_key: "/Users/yourname/.ssh/id_rsa" # 使用私钥登录时填入绝对路径
                python_interpreter: "/usr/bin/python3"         # 远程 Python 解释器路径
    ```

### 1.2 多环境配置与切换

Ansible 命令组支持多环境配置，通过 `-e` 或 `--env` 参数指定环境：

1. **测试环境配置**：
   复制为 `configs/ansible.test.yaml` 并填写测试机资产：
   ```bash
   cp configs/ansible.example.yaml configs/ansible.test.yaml
   ```
   使用 `-e test` 执行命令：
   ```bash
   opsvault ansible ping -e test --group db_servers
   ```

2. **生产环境配置**：
   复制为 `configs/ansible.prod.yaml` 并填写生产机资产：
   ```bash
   cp configs/ansible.example.yaml configs/ansible.prod.yaml
   ```
   使用 `-e prod` 执行命令：
   ```bash
   opsvault ansible ping -e prod --group db_servers
   ```

> [!IMPORTANT]
> **关于 macOS 控制端的写权限说明**：
> 默认的临时目录如果配置为 `/data/opsvault/ansible/tmp`，而在 macOS 上根目录 `/data` 可能是只读的，会导致报错：`mkdir /data: read-only file system`。
> **解决方法**：请在对应的配置文件（如 `configs/ansible.yaml`）中，将 `ansible.temp_dir` 修改为本地可写目录（如 `./ansible_tmp` 或 `/tmp/ansible`）。

### 2. 批量运维命令示例

*   **批量连通性测试 (Ping)**：
    ```bash
    # 测试所有主机的 SSH 连通性
    opsvault ansible ping --group my_servers
    ```
*   **批量执行临时 Shell 命令 (Exec)**：
    ```bash
    # 在受控机器上并发执行 ad-hoc 命令
    opsvault ansible exec --cmd "uptime" --group my_servers
    ```
*   **查看主机与分组清单 (List)**：
    快速查看当前 Ansible 配置文件（如 `ansible.yaml` 或 `-e prod` 对应环境）中已配置的所有服务器分组列表、目标主机及连接认证方式：
    ```bash
    opsvault ansible list
    ```
*   **多机环境巡检 (Doctor)**：
    获取远程节点的负载、内存、磁盘以及 Docker/Nginx 服务运行状态，并在终端渲染出 Lipgloss 漂亮的统计表格：
    ```bash
    opsvault ansible doctor --group my_servers
    ```
*   **批量一键部署中间件 (Deploy)**：
    通过 Ansible Playbook 批量向远程机器部署指定中间件（支持 docker、mysql、redis、rabbitmq、nginx、minio）：
    ```bash
    # 远程批量拉取并部署 Redis 容器，包含专属网桥与持久化挂载
    opsvault ansible deploy --service redis --group my_servers
    ```
    > [!NOTE]
    > 当部署 `mysql`、`redis`、`rabbitmq`、`minio` 时如果密码为空，系统会自动通过 `credutil.GenPassword(20)` 生成 20 位高强度密码下发至集群，并在终端通过 Lipgloss 彩色边框卡片清晰输出部署成功凭据。
*   **二进制与配置一键下发 (Push)**：
    将控制端本地编译好的 Linux 二进制文件（如 `opsvault-linux-amd64`）和 `default.yaml` 配置文件推送至被控集群主机的 `/data/opsvault/bin` 和 `/configs` 中，并自动在目标服务器创建 `/usr/local/bin/opsvault` 软链接。
    ```bash
    # 下发可执行文件与配置文件到被控端
    opsvault ansible push --group my_servers --bin ./bin/opsvault-linux-amd64 --config-path ./configs/default.yaml
    ```
    **✨ 边缘协同亮点**：下发成功后，登录任意目标节点 SSH，敲下 `opsvault tui` 或 `opsvault status` 即可就地使用可视化运维终端，实现“中心批量自动化，边缘单机深度排查”！
*   **批量优雅回收与清理 (Uninstall)**：
    对远程集群批量卸载指定组件或 Docker Engine。携带 `--purge` 标志时会彻底物理擦除挂载数据卷与站点日志目录：
    ```bash
    # 仅卸载服务容器/进程（安全保留数据卷）
    opsvault ansible uninstall --service mysql --group my_servers

    # 深度物理回收（连同宿主机 /data/opsvault/rabbitmq 挂载目录彻底清空）
    opsvault ansible uninstall --service rabbitmq --group my_servers --purge
    ```

## TUI 运维工作台说明

使用 `opsvault tui` 命令即可启动全交互式的 OpsVault 运维控制台。

### 核心区域与焦点切换

- 控制台拥有三个焦点区域：**Sidebar（左侧边栏）**、**Detail（右侧详情区）**、**Drawer（底部抽屉）**。
- 按下 **`Tab`** 键可在当前开启的焦点区域之间循环切换焦点。

### 全局导航

- 按 **`h` / `l`** 或 **`←` / `→`** 键可在顶部的 `Dashboard`、`Nginx`、`Docker`、`Config` 面板页签之间切换。
- 在 `Dashboard` 面板的高亮行按下 **`Enter`** 可以直接跳转到对应的子管理面板。
- 按 **`q`** 或 **`Ctrl+C`** 退出控制台。

### 高频操作快捷键

在 `Dashboard` 或 `Docker`/`Nginx` 服务面板，当选中具体服务后，可通过以下单键快捷执行操作：

- **`s`**：启动服务 (Start)
- **`x`**：停止服务 (Stop)
- **`r`**：重启服务 (Restart)
- **`i`**：安装服务 (Install)
- **`l`**：在底部抽屉查看最新日志 (Logs)
- **`t`**：切换底部抽屉，显示最近执行任务的状态与输出 (Task Logs)
- **`d`**：卸载服务 / 删除虚拟主机 (Uninstall / Delete) — 属于**危险操作**，会触发红色的二次确认提示
- **`esc`**：取消当前确认提示或折叠底部抽屉

### 资源管理快捷键 (Nginx 面板)

在 Nginx 子面板的 `VHosts` 与 `Certificates` 子模式中：

- **VHosts（虚拟主机管理）**：
  - **`a`**：添加新的虚拟主机。交互式提示您输入域名（Domain Name），按回车后系统会自动推荐 Root 目录，按回车或修改后回车即可一键生成配置并自动热重载 Nginx。
  - **`d`**：删除所选中的虚拟主机配置文件（安全起见，默认保留网站根目录文件），需要输入 `y` 或 `Enter` 二次确认。
- **Certificates（SSL 证书管理）**：
  - **`a`**：输入域名为特定站点申请 Let's Encrypt 证书并绑定启用 HTTPS。
  - **`r`**：为当前选中的站点手动续期 SSL 证书。
  - **`d`**：删除 SSL 证书并恢复 HTTP 状态。

