# TUI Operations Workbench Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the current read-only OpsVault TUI into an interactive operations workbench for Docker services and Nginx high-frequency workflows.

**Architecture:** Keep `driver.ServiceDriver` unchanged as the fixed lifecycle contract, then layer optional capability interfaces on top for logs, reload, Nginx vhost management, and Nginx SSL management. Refactor `internal/tui` from string-only panel helpers into an interactive shell with panel state, action dispatch, and a bottom drawer for tasks and logs.

**Tech Stack:** Go, Bubble Tea, Lipgloss, Cobra, Viper, Docker SDK, existing OpsVault driver packages

---

### Task 1: Add driver capability interfaces and log entry points

**Files:**
- Modify: `internal/driver/driver.go`
- Modify: `internal/driver/docker/base.go`
- Modify: `internal/driver/binary/nginx.go`
- Modify: `internal/driver/docker/specs_test.go`
- Modify: `internal/driver/binary/nginx_test.go`

- [ ] **Step 1: Write the failing tests for log access and optional capabilities**

Add assertions that prove:
- Docker drivers can return recent logs through a driver-layer method instead of only Cobra commands.
- `NginxDriver` exposes reload, vhost, SSL, and log behavior through driver-facing methods.

Use table-driven tests in:
- `internal/driver/docker/specs_test.go`
- `internal/driver/binary/nginx_test.go`

Target shapes to test:

```go
type LogReader interface {
    TailLogs(lines int) (string, error)
}

type Reloadable interface {
    Reload() error
}
```

- [ ] **Step 2: Run the focused tests and confirm failure is from missing methods**

Run:

```bash
go test ./internal/driver/docker ./internal/driver/binary
```

Expected:
- build or test failures complaining about missing log/capability methods

- [ ] **Step 3: Implement minimal capability interfaces in `internal/driver/driver.go`**

Add optional interfaces without changing `ServiceDriver`:

```go
type LogReader interface {
    TailLogs(lines int) (string, error)
}

type Reloadable interface {
    Reload() error
}

type VHostManager interface {
    AddVHost(domain, root string) error
    DeleteVHost(domain string, deleteRoot bool) error
    ListVHosts() ([]map[string]string, error)
}

type SSLManager interface {
    ApplySSL(domain string) error
    RenewSSL(domain string) error
    DeleteSSL(domain string) error
}
```

Keep names and signatures aligned with the approved spec. Do not widen `ServiceDriver`.

- [ ] **Step 4: Move log behavior and Nginx SSL orchestration into the driver layer**

Implement:
- `BaseDriver.TailLogs(lines int) (string, error)` by calling Docker logs and collecting stdout/stderr output
- `NginxDriver.TailLogs(lines int) (string, error)` by running `journalctl -u nginx -n <lines> --no-pager`
- `NginxDriver.ApplySSL(domain string) error` by calling `sslutil.Manager.Apply(...)` and then `EnableSSL(domain)`
- `NginxDriver.RenewSSL(domain string) error` by calling `sslutil.Manager.Renew(domain)` and then `Reload()`
- `NginxDriver.DeleteSSL(domain string) error` by calling `sslutil.Manager.Delete(domain)` and then `DisableSSL(domain)`

Leave Cobra as a thin caller after this step.

- [ ] **Step 5: Re-run focused tests and keep all driver tests green**

Run:

```bash
go test ./internal/driver/docker ./internal/driver/binary
```

Expected:
- PASS for both packages

- [ ] **Step 6: Commit the driver capability layer**

```bash
git add internal/driver/driver.go internal/driver/docker/base.go internal/driver/binary/nginx.go internal/driver/docker/specs_test.go internal/driver/binary/nginx_test.go
git commit -m "feat(driver): add tui capability interfaces and log support"
```

### Task 2: Introduce TUI service registry, actions, and task message types

**Files:**
- Create: `internal/tui/actions.go`
- Create: `internal/tui/task_manager.go`
- Modify: `internal/tui/status_provider.go`
- Modify: `internal/tui/root_model_test.go`

- [ ] **Step 1: Write failing tests for service registry output and task message flow**

Add tests covering:
- service definitions returned for nginx, mysql, redis, rocketmq, rabbitmq, postgres
- action availability derived from status and capability interfaces
- task lifecycle messages such as started/succeeded/failed

Representative shapes:

```go
type ServiceRef struct {
    Name   string
    Driver driver.ServiceDriver
}

type ActionID string

const (
    ActionInstall ActionID = "install"
    ActionStart   ActionID = "start"
    ActionStop    ActionID = "stop"
)
```

- [ ] **Step 2: Run the TUI tests and confirm failure comes from missing registry/action types**

Run:

```bash
go test ./internal/tui
```

Expected:
- compile or test failures for missing action and task-manager types

- [ ] **Step 3: Implement `actions.go` with explicit action IDs and action resolution**

Create a small, deterministic action layer:

```go
type Action struct {
    ID        ActionID
    Label     string
    Dangerous bool
}

func AvailableServiceActions(status driver.ServiceStatus, svc ServiceRef) []Action
```

Rules:
- running services expose `stop`, `restart`, `logs`
- stopped installed services expose `start`, `logs`
- uninstalled services expose `install`
- services supporting upgrade expose `upgrade`
- uninstall remains available after install and is marked dangerous
- nginx service adds `reload` when `Reloadable`

- [ ] **Step 4: Implement `task_manager.go` with Bubble Tea-compatible task messages**

Add:

```go
type taskStartedMsg struct { ... }
type taskFinishedMsg struct { ... }
type taskFailedMsg struct { ... }
type taskOutputMsg struct { ... }
```

Add a small runner:

```go
func runAction(service ServiceRef, action Action, params map[string]string) tea.Cmd
```

For this phase, output can be one final string plus error status. Continuous streaming can stay out of scope.

- [ ] **Step 5: Refactor `status_provider.go` into a service registry builder**

Keep `Statuses()` support for refresh, but also add a way to build stable service references using the same driver constructors so the TUI can:
- read statuses
- discover capabilities
- run actions

Do not duplicate driver construction in multiple TUI files after this point.

- [ ] **Step 6: Re-run TUI tests**

Run:

```bash
go test ./internal/tui
```

Expected:
- PASS with new registry and task-manager coverage

- [ ] **Step 7: Commit the service/action foundation**

```bash
git add internal/tui/actions.go internal/tui/task_manager.go internal/tui/status_provider.go internal/tui/root_model_test.go
git commit -m "feat(tui): add service registry and action model"
```

### Task 3: Refactor the root TUI shell into an interactive workbench

**Files:**
- Modify: `internal/tui/root_model.go`
- Create: `internal/tui/drawer.go`
- Modify: `internal/tui/dashboard.go`
- Modify: `internal/tui/root_model_test.go`

- [ ] **Step 1: Write failing tests for focus management, tab switching, and drawer behavior**

Add tests for:
- `Tab` cycling focus regions
- left/right tab switching preserved from current behavior
- drawer open/close on task/log commands
- refresh messages updating global status state without losing selection

Suggested focus enum:

```go
type focusRegion int

const (
    focusSidebar focusRegion = iota
    focusDetail
    focusDrawer
)
```

- [ ] **Step 2: Run the focused root model tests**

Run:

```bash
go test ./internal/tui -run 'TestRootModel'
```

Expected:
- failures around missing focus and drawer behavior

- [ ] **Step 3: Replace string-only shell behavior with a structured root model**

Refactor `RootModel` to own:
- active tab
- focus region
- active panel instances
- shared service statuses
- selected task/log drawer state
- global last error

Keep periodic refresh via `tickRefresh()`.

- [ ] **Step 4: Add `drawer.go` for bottom drawer rendering**

Implement a compact drawer with three modes:

```go
type drawerMode int

const (
    drawerHidden drawerMode = iota
    drawerTasks
    drawerLogs
    drawerOutput
)
```

First release requirements:
- show recent task result lines
- show current log payload
- show current task output payload

- [ ] **Step 5: Keep Dashboard usable as a summary-and-jump surface**

Update `dashboard.go` so it renders a selectable services table or list summary instead of a static text block. It should support at least:
- current service highlight
- display of high-frequency actions hint
- jump target into Nginx or Docker tabs

Avoid putting long forms here.

- [ ] **Step 6: Re-run TUI tests**

Run:

```bash
go test ./internal/tui
```

Expected:
- PASS with root model and drawer assertions

- [ ] **Step 7: Commit the workbench shell**

```bash
git add internal/tui/root_model.go internal/tui/drawer.go internal/tui/dashboard.go internal/tui/root_model_test.go
git commit -m "feat(tui): build interactive workbench shell"
```

### Task 4: Implement the Docker panel as the first full operations panel

**Files:**
- Modify: `internal/tui/docker_panel.go`
- Modify: `internal/tui/root_model.go`
- Modify: `internal/tui/actions.go`
- Modify: `internal/tui/root_model_test.go`

- [ ] **Step 1: Write failing tests for Docker panel selection and action dispatch**

Cover:
- moving between docker services
- action list changes based on status
- dangerous uninstall action requiring confirmation
- `logs` action filling the drawer with log content

Example checks:

```go
if !containsAction(actions, ActionStart) { ... }
if !containsAction(actions, ActionUninstall) { ... }
```

- [ ] **Step 2: Run focused Docker panel tests**

Run:

```bash
go test ./internal/tui -run 'TestDocker|TestRootModel'
```

Expected:
- failures for missing Docker panel interaction behavior

- [ ] **Step 3: Replace `DockerPanelView` with a stateful panel model**

Add a panel struct that owns:
- selected service index
- current confirmation state
- selected action index or hotkey dispatch
- current panel-local error

It should render:
- left rail service list
- right detail block with status/image/network/ports/data path
- action row with install/start/stop/restart/upgrade/uninstall/logs

- [ ] **Step 4: Wire action hotkeys and confirmation flow**

Support the approved first-pass keys:
- `s` start
- `x` stop
- `r` restart
- `u` upgrade
- `i` install
- `d` dangerous action confirm
- `l` logs
- `Enter` primary action
- `esc` close confirm or drawer

For uninstall:
- first keypress enters confirmation
- second confirmation executes task

- [ ] **Step 5: Refresh state after every Docker action**

Ensure successful actions trigger:
- task result in drawer
- automatic status reload
- preserved service selection where practical

- [ ] **Step 6: Run TUI tests and package-level regression tests**

Run:

```bash
go test ./internal/tui ./internal/driver/...
```

Expected:
- PASS across TUI and drivers

- [ ] **Step 7: Commit the Docker operations panel**

```bash
git add internal/tui/docker_panel.go internal/tui/root_model.go internal/tui/actions.go internal/tui/root_model_test.go
git commit -m "feat(tui): add interactive docker operations panel"
```

### Task 5: Implement Nginx service, vhost, and SSL workbench flows

**Files:**
- Modify: `internal/tui/nginx_panel.go`
- Modify: `internal/tui/root_model.go`
- Modify: `internal/tui/actions.go`
- Modify: `internal/tui/root_model_test.go`
- Modify: `cmd/nginx/ssl.go`
- Modify: `cmd/nginx/log.go`
- Modify: `cmd/rocketmq/log.go`
- Modify: `cmd/mysql/log.go`

- [ ] **Step 1: Write failing tests for Nginx panel modes and form submissions**

Cover:
- switching among `Service`, `VHosts`, and `Certificates`
- service actions including `reload`
- vhost add/delete flow
- SSL apply/renew/delete flow
- log action opening the drawer

Use small form-state tests such as:

```go
type nginxMode int

const (
    nginxModeService nginxMode = iota
    nginxModeVHosts
    nginxModeCertificates
)
```

- [ ] **Step 2: Run focused Nginx tests**

Run:

```bash
go test ./internal/tui -run 'TestNginx|TestRootModel'
```

Expected:
- failures for missing Nginx workbench behavior

- [ ] **Step 3: Replace `NginxPanelView` with a stateful resource panel**

The panel should support:
- service details and actions when in service mode
- list/detail rendering for vhosts
- list/detail rendering for certificates using minimal metadata if necessary

Keep first-pass forms intentionally small:
- vhost add: `domain`, `root`
- vhost delete: `domain`, optional `delete-root`
- SSL apply/delete: `domain`
- SSL renew: optional `domain`, empty means all

- [ ] **Step 4: Move Cobra log and SSL commands to the new driver methods**

Update command handlers so they call the same driver methods used by TUI:
- Nginx CLI uses `ApplySSL`, `RenewSSL`, `DeleteSSL`, `TailLogs`
- Docker-backed log commands use `TailLogs`

This keeps command and TUI behavior aligned.

- [ ] **Step 5: Add confirmation handling for Nginx dangerous actions**

Confirm at least:
- vhost delete with `delete-root`
- SSL delete
- uninstall and purge uninstall

- [ ] **Step 6: Re-run TUI and command/driver regression tests**

Run:

```bash
go test ./internal/tui ./internal/driver/... ./cmd/...
```

Expected:
- PASS across TUI, driver, and command packages

- [ ] **Step 7: Commit the Nginx workbench flows**

```bash
git add internal/tui/nginx_panel.go internal/tui/root_model.go internal/tui/actions.go internal/tui/root_model_test.go cmd/nginx/ssl.go cmd/nginx/log.go cmd/rocketmq/log.go cmd/mysql/log.go
git commit -m "feat(tui): add nginx workbench operations"
```

### Task 6: Finish integration, polish refresh behavior, and verify end to end

**Files:**
- Modify: `cmd/tui.go`
- Modify: `internal/tui/config_wizard.go`
- Modify: `README.md`
- Modify: `internal/tui/status_provider.go`
- Modify: `internal/tui/root_model_test.go`

- [ ] **Step 1: Write any final failing regression tests for integration edges**

Focus on:
- `opsvault tui` bootstrapping the runtime registry cleanly
- refresh behavior when Docker client creation fails
- placeholder config panel still rendering without blocking navigation

- [ ] **Step 2: Run focused integration tests**

Run:

```bash
go test ./internal/tui ./cmd -run 'TestRoot|TestTUI'
```

Expected:
- failures only for missing integration polish

- [ ] **Step 3: Finalize bootstrapping and placeholder surfaces**

Ensure:
- `cmd/tui.go` builds the new provider/registry cleanly
- status refresh errors stay visible but non-fatal
- config panel remains a placeholder rather than dead code

- [ ] **Step 4: Update README with the new interactive TUI scope**

Document:
- current TUI workbench capabilities
- first-release limits
- key bindings for high-frequency actions

- [ ] **Step 5: Run formatting and full verification**

Run:

```bash
gofmt -w ./cmd ./internal
go test ./...
go build ./...
```

Expected:
- `gofmt` produces no follow-up edits after rerun
- `go test ./...` passes
- `go build ./...` succeeds

- [ ] **Step 6: Commit the integrated workbench**

```bash
git add cmd/tui.go internal/tui/config_wizard.go README.md internal/tui/status_provider.go internal/tui/root_model_test.go
git commit -m "feat(tui): ship first interactive operations workbench"
```

## Execution Notes

- Keep `Config` as a placeholder panel for this milestone.
- Do not introduce streaming log infrastructure in the first pass.
- Do not widen `ServiceDriver`.
- Prefer small helpers over a large shared panel abstraction unless duplication becomes real.
- Preserve current passing package tests while growing coverage around TUI behavior and driver capabilities.
