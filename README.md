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
