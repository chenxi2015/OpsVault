# OpsVault Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first working OpsVault foundation with a complete Cobra command tree, driver abstractions, Docker and OneinStack integration scaffolding, and a minimal Bubble Tea TUI that compiles and is test-covered.

**Architecture:** Commands stay thin and only translate flags into driver calls. Runtime behavior lives under `internal/driver`, `internal/oneinstack`, `internal/system`, and `pkg/*` helper packages. Docker lifecycle operations are centralized in `pkg/dockercli`, and TUI screens consume the same driver-facing service abstractions instead of duplicating deployment logic.

**Tech Stack:** Go, Cobra, Viper, Docker SDK, Bubble Tea, Bubbles, Lipgloss

---

### Task 1: Add tests for core config and helper behavior

**Files:**
- Create: `pkg/netutil/cidr_test.go`
- Create: `internal/oneinstack/download_test.go`
- Create: `cmd/root_test.go`

- [ ] **Step 1: Write the failing tests for CIDR validation, OneinStack URL extraction, and root config bootstrap**
- [ ] **Step 2: Run `go test ./...` and confirm failures come from missing implementation**
- [ ] **Step 3: Implement the minimal code required for the tests to pass**
- [ ] **Step 4: Re-run `go test ./...` and keep the suite green**

### Task 2: Build config, logging, and driver foundation

**Files:**
- Modify: `cmd/root.go`
- Create: `configs/default.yaml`
- Create: `pkg/logger/logger.go`
- Create: `pkg/netutil/cidr.go`
- Create: `pkg/fileutil/fileutil.go`
- Create: `pkg/dockercli/client.go`
- Create: `internal/driver/driver.go`
- Create: `internal/driver/docker/base.go`
- Create: `internal/driver/docker/network.go`
- Create: `internal/driver/docker/mysql.go`
- Create: `internal/driver/docker/redis.go`
- Create: `internal/driver/docker/rocketmq.go`
- Create: `internal/driver/docker/rabbitmq.go`
- Create: `internal/driver/docker/postgres.go`
- Create: `internal/driver/binary/base.go`
- Create: `internal/driver/binary/nginx.go`
- Create: `internal/oneinstack/download.go`
- Create: `internal/oneinstack/runner.go`
- Create: `internal/system/port.go`
- Create: `internal/system/proc.go`
- Create: `internal/system/sysctl.go`
- Create: `internal/system/systemd.go`

- [ ] **Step 1: Implement viper-backed root initialization and global flags**
- [ ] **Step 2: Implement shared status types, service metadata, and driver constructors**
- [ ] **Step 3: Implement Docker network helpers and lifecycle helpers via Docker SDK**
- [ ] **Step 4: Implement Binary Nginx driver and OneinStack script wrappers**
- [ ] **Step 5: Run focused tests, then `go test ./...`**

### Task 3: Build the Cobra command tree for all required services

**Files:**
- Modify: `main.go`
- Create: `cmd/tui.go`
- Create: `cmd/nginx/root.go`
- Create: `cmd/nginx/install.go`
- Create: `cmd/nginx/start.go`
- Create: `cmd/nginx/stop.go`
- Create: `cmd/nginx/restart.go`
- Create: `cmd/nginx/uninstall.go`
- Create: `cmd/nginx/upgrade.go`
- Create: `cmd/nginx/vhost.go`
- Create: `cmd/nginx/ssl.go`
- Create: `cmd/nginx/status.go`
- Create: `cmd/nginx/log.go`
- Create: `cmd/mysql/root.go`
- Create: `cmd/mysql/install.go`
- Create: `cmd/mysql/start.go`
- Create: `cmd/mysql/stop.go`
- Create: `cmd/mysql/restart.go`
- Create: `cmd/mysql/uninstall.go`
- Create: `cmd/mysql/upgrade.go`
- Create: `cmd/mysql/status.go`
- Create: `cmd/mysql/log.go`
- Create: `cmd/redis/root.go`
- Create: `cmd/redis/install.go`
- Create: `cmd/redis/start.go`
- Create: `cmd/redis/stop.go`
- Create: `cmd/redis/restart.go`
- Create: `cmd/redis/uninstall.go`
- Create: `cmd/redis/upgrade.go`
- Create: `cmd/redis/status.go`
- Create: `cmd/rocketmq/root.go`
- Create: `cmd/rocketmq/install.go`
- Create: `cmd/rocketmq/start.go`
- Create: `cmd/rocketmq/stop.go`
- Create: `cmd/rocketmq/restart.go`
- Create: `cmd/rocketmq/uninstall.go`
- Create: `cmd/rocketmq/upgrade.go`
- Create: `cmd/rocketmq/version.go`
- Create: `cmd/rocketmq/dlq.go`
- Create: `cmd/rocketmq/log.go`
- Create: `cmd/rabbitmq/root.go`
- Create: `cmd/rabbitmq/install.go`
- Create: `cmd/rabbitmq/start.go`
- Create: `cmd/rabbitmq/stop.go`
- Create: `cmd/rabbitmq/restart.go`
- Create: `cmd/rabbitmq/uninstall.go`
- Create: `cmd/rabbitmq/upgrade.go`
- Create: `cmd/rabbitmq/status.go`
- Create: `cmd/postgres/root.go`
- Create: `cmd/postgres/install.go`
- Create: `cmd/postgres/start.go`
- Create: `cmd/postgres/stop.go`
- Create: `cmd/postgres/uninstall.go`
- Create: `cmd/postgres/upgrade.go`

- [ ] **Step 1: Add each service root command and register it from `cmd/root.go`**
- [ ] **Step 2: Wire lifecycle commands to the correct Docker or Binary driver**
- [ ] **Step 3: Add Nginx vhost/SSL/status/log subcommands**
- [ ] **Step 4: Add RocketMQ `version` and `dlq stat` placeholders backed by driver/system wrappers**
- [ ] **Step 5: Run `go test ./...` and `go test ./... -run TestRoot` as a smoke check**

### Task 4: Build the first runnable Bubble Tea TUI

**Files:**
- Create: `internal/tui/root_model.go`
- Create: `internal/tui/dashboard.go`
- Create: `internal/tui/nginx_panel.go`
- Create: `internal/tui/docker_panel.go`
- Create: `internal/tui/config_wizard.go`

- [ ] **Step 1: Implement the root Bubble Tea model and top-level navigation**
- [ ] **Step 2: Render dashboard, Nginx panel, Docker panel, and config wizard views**
- [ ] **Step 3: Wire `opsvault tui` to launch the model**
- [ ] **Step 4: Run `go test ./...` and `go build ./...`**

### Task 5: Finish documentation and verify the branch

**Files:**
- Modify: `README.md`
- Modify: `go.mod`

- [ ] **Step 1: Update dependencies and document the current foundation behavior**
- [ ] **Step 2: Run `gofmt -w` on touched Go files**
- [ ] **Step 3: Run `go test ./...`**
- [ ] **Step 4: Run `go build ./...`**
