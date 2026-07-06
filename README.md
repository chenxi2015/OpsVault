# OpsVault

OpsVault 是一个面向 CentOS 7 / CentOS Stream 的运维工具箱，提供：

- 基于 `cobra` + `viper` 的分层 CLI
- 基于 `bubbletea` + `lipgloss` 的 TUI 入口
- 基于 Docker SDK 的中间件驱动骨架
- 内建源码编译的 Nginx 二进制驱动骨架

## 当前已完成

- 完整命令树：
  - `opsvault tui`
  - `opsvault nginx ...`
  - `opsvault mysql ...`
  - `opsvault redis ...`
  - `opsvault rocketmq ...`
  - `opsvault rabbitmq ...`
  - `opsvault postgres ...`
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

