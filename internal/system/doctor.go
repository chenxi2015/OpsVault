package system

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"OpsVault/pkg/dockercli"
	"OpsVault/pkg/netutil"
	"OpsVault/pkg/sysutil"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/spf13/viper"
)

// DiagnosticStatus represents the status of a single diagnostic check.
type DiagnosticStatus string

const (
	StatusOk   DiagnosticStatus = "OK"
	StatusWarn DiagnosticStatus = "WARN"
	StatusFail DiagnosticStatus = "FAIL"
)

// DiagnosticItem holds the result of a single check.
type DiagnosticItem struct {
	Name       string           `json:"name"`
	Status     DiagnosticStatus `json:"status"`
	Message    string           `json:"message"`
	Suggestion string           `json:"suggestion,omitempty"`
}

// RunDiagnostics executes all system checks and returns a list of results.
func RunDiagnostics(ctx context.Context, config *viper.Viper, dockerCli *client.Client) ([]DiagnosticItem, error) {
	var items []DiagnosticItem

	// 1. OS Compatibility Check
	items = append(items, checkOS())

	// 2. User Privilege Check
	items = append(items, checkPrivilege())

	// 3. Storage Directory Writable Check
	items = append(items, checkStorageWritable(config))

	// 4. Docker Daemon Status Check
	dockerOk, dockerItem, resolvedCli := checkDockerDaemon(ctx, dockerCli)
	items = append(items, dockerItem)

	// 5. Docker Network Check
	if dockerOk && resolvedCli != nil {
		items = append(items, checkDockerNetwork(ctx, config, resolvedCli))
	}

	// 6. Port Availability Check
	items = append(items, checkPorts(config))

	// 7. Nginx Compilation Tools Check
	items = append(items, checkCompilationTools())

	// 8. File Descriptor Limits Check
	items = append(items, checkFileLimits())

	return items, nil
}

func checkOS() DiagnosticItem {
	item := DiagnosticItem{Name: "操作系统兼容性"}
	if !sysutil.IsLinux() {
		item.Status = StatusWarn
		item.Message = fmt.Sprintf("当前系统为 %s, OpsVault 仅正式支持 CentOS 7 / CentOS Stream。", runtime.GOOS)
		item.Suggestion = "建议在 CentOS 7 或 CentOS Stream 虚拟机/服务器环境运行以获得最佳兼容性。"
		return item
	}

	// Read os-release to verify CentOS
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		// Fallback to redhat-release
		data, err = os.ReadFile("/etc/redhat-release")
	}

	if err == nil {
		content := strings.ToLower(string(data))
		if strings.Contains(content, "centos") || strings.Contains(content, "red hat") || strings.Contains(content, "rocky") || strings.Contains(content, "almalinux") {
			item.Status = StatusOk
			// Extract first line for display
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					pretty := strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
					item.Message = fmt.Sprintf("兼容的 Linux 系统 (%s)", pretty)
					return item
				}
			}
			item.Message = "兼容的 RedHat/CentOS 家族 Linux system"
			return item
		}
	}

	item.Status = StatusWarn
	item.Message = "未检测到 CentOS/RedHat 家族系统，部分二进制源码编译及脚本可能无法正常工作。"
	item.Suggestion = "建议使用 CentOS 7/CentOS Stream/Rocky Linux。"
	return item
}

func checkPrivilege() DiagnosticItem {
	item := DiagnosticItem{Name: "用户运行权限"}
	if sysutil.IsRoot() {
		item.Status = StatusOk
		item.Message = "当前以 root 用户运行"
	} else {
		item.Status = StatusFail
		item.Message = "当前用户不是 root 用户 (UID != 0)"
		item.Suggestion = "请使用 sudo 运行程序，或切换到 root 用户来执行运维操作。"
	}
	return item
}

func checkStorageWritable(config *viper.Viper) DiagnosticItem {
	item := DiagnosticItem{Name: "持久化存储根目录"}
	dataRoot := "/data/opsvault"
	if config != nil && config.GetString("docker.data_root") != "" {
		dataRoot = config.GetString("docker.data_root")
	}

	writable, err := isWritableOrParentWritable(dataRoot)
	if err != nil || !writable {
		item.Status = StatusFail
		item.Message = fmt.Sprintf("存储根目录 %s (或其可写父目录) 无法写入", dataRoot)
		if err != nil {
			item.Message += fmt.Sprintf(": %v", err)
		}
		item.Suggestion = "请确认磁盘未写满、未挂载为只读文件系统，且当前用户有权限在 /data 下创建目录。"
	} else {
		item.Status = StatusOk
		item.Message = fmt.Sprintf("存储根目录 %s 检查通过 (可读写)", dataRoot)
	}
	return item
}

func isWritableOrParentWritable(path string) (bool, error) {
	current := path
	for {
		info, err := os.Stat(current)
		if err == nil {
			if !info.IsDir() {
				return false, fmt.Errorf("%s exists but is not a directory", current)
			}
			// Directory exists, test writability by creating a temporary file
			tmpFile := filepath.Join(current, ".opsvault_write_test")
			f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return false, nil
			}
			f.Close()
			_ = os.Remove(tmpFile)
			return true, nil
		}

		if !os.IsNotExist(err) {
			return false, err
		}

		// Directory does not exist, check parent
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return false, nil
}

func checkDockerDaemon(ctx context.Context, dockerCli *client.Client) (bool, DiagnosticItem, *client.Client) {
	item := DiagnosticItem{Name: "Docker 运行状态"}

	// Check if docker binary exists
	_, err := exec.LookPath("docker")
	if err != nil {
		item.Status = StatusFail
		item.Message = "未在系统 PATH 中找到 docker 命令"
		item.Suggestion = "请在宿主机上安装 Docker。可以通过官方脚本 'curl -fsSL https://get.docker.com | bash' 一键安装。"
		return false, item, nil
	}

	cli := dockerCli
	if cli == nil {
		var err error
		cli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			item.Status = StatusFail
			item.Message = fmt.Sprintf("无法初始化 Docker 客户端: %v", err)
			item.Suggestion = "请检查 Docker 环境变量，或确认 Docker 是否正确安装。"
			return false, item, nil
		}
	}

	// Ping Docker daemon
	_, err = cli.Ping(ctx)
	if err != nil {
		item.Status = StatusFail
		item.Message = fmt.Sprintf("无法连接到 Docker 守护进程: %v", err)
		item.Suggestion = "请确认 Docker 服务已启动 (如运行 'systemctl start docker')，且当前用户有权访问 /var/run/docker.sock。"
		return false, item, nil
	}

	item.Status = StatusOk
	item.Message = "Docker 守护进程连接正常"
	return true, item, cli
}

func checkDockerNetwork(ctx context.Context, config *viper.Viper, cli *client.Client) DiagnosticItem {
	item := DiagnosticItem{Name: "Docker 专属网桥"}
	netName := dockercli.ResolveNetworkName(config)
	cidr := "172.28.0.0/16"
	if config != nil {
		if val := config.GetString("docker.cidr"); val != "" {
			cidr = val
		}
	}

	// Validate CIDR first
	if err := netutil.ValidateCIDR(cidr); err != nil {
		item.Status = StatusFail
		item.Message = fmt.Sprintf("配置文件中定义的 CIDR 不合法: %v", err)
		item.Suggestion = "请在 configs/default.yaml 中修改 'docker.cidr' 为合法的子网网段（例如 172.28.0.0/16）。"
		return item
	}

	existing, err := cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		item.Status = StatusWarn
		item.Message = fmt.Sprintf("无法列出 Docker 网桥: %v", err)
		item.Suggestion = "请确认 Docker 守护进程未死锁，且网络驱动正常。"
		return item
	}

	for _, nw := range existing {
		if nw.Name == netName {
			item.Status = StatusOk
			item.Message = fmt.Sprintf("专属网桥 %s 已存在", netName)
			return item
		}
	}

	item.Status = StatusOk
	item.Message = fmt.Sprintf("专属网桥 %s 尚未创建，将在首次部署服务容器时自动创建", netName)
	return item
}

func checkPorts(config *viper.Viper) DiagnosticItem {
	item := DiagnosticItem{Name: "关键运维端口占用"}

	type PortInfo struct {
		Service   string
		ConfigKey string
		Default   int
	}

	portsToCheck := []PortInfo{
		{Service: "Nginx HTTP", ConfigKey: "nginx (默认)", Default: 80},
		{Service: "Nginx HTTPS", ConfigKey: "nginx (默认)", Default: 443},
		{Service: "MySQL 数据库", ConfigKey: "mysql.port", Default: 3306},
		{Service: "Redis 缓存", ConfigKey: "redis.port", Default: 6379},
		{Service: "RocketMQ NameServer", ConfigKey: "rocketmq.namesrv_port", Default: 9876},
		{Service: "RocketMQ Broker", ConfigKey: "rocketmq.broker_port", Default: 10911},
		{Service: "RabbitMQ 消息队列", ConfigKey: "rabbitmq.port", Default: 5672},
		{Service: "RabbitMQ Web 控制台", ConfigKey: "rabbitmq.ui_port", Default: 15672},
		{Service: "PostgreSQL 数据库", ConfigKey: "postgres.port", Default: 5432},
	}

	type Row struct {
		Service    string
		ConfigPort int
		ConfigKey  string
		Status     string
		Occupant   string
	}

	var rows []Row
	hasOccupied := false
	var occupiedNames []string

	for _, pt := range portsToCheck {
		port := pt.Default
		configKeyDisplay := pt.ConfigKey
		if config != nil && pt.ConfigKey != "nginx (默认)" {
			if val := config.GetInt(pt.ConfigKey); val != 0 {
				port = val
			}
		}

		statusStr := "未使用"
		occupantStr := "-"

		if err := CheckPortAvailable(port); err != nil {
			statusStr = "已占用"
			hasOccupied = true
			occupiedNames = append(occupiedNames, fmt.Sprintf("%d (%s)", port, pt.Service))

			// Try to find the process occupying the port
			if pid, procName, err := GetPortOccupant(port); err == nil {
				occupantStr = fmt.Sprintf("%d/%s", pid, procName)
			} else {
				occupantStr = "未知进程"
			}
		}

		rows = append(rows, Row{
			Service:    pt.Service,
			ConfigPort: port,
			ConfigKey:  configKeyDisplay,
			Status:     statusStr,
			Occupant:   occupantStr,
		})
	}

	// Build tabular output
	var tableLines []string
	col1Width := 20 // 服务名称
	col2Width := 8  // 配置端口
	col3Width := 22 // 配置来源
	col4Width := 10 // 当前状态
	col5Width := 18 // 占用进程

	// Top border
	tableLines = append(tableLines, fmt.Sprintf("+%s+%s+%s+%s+%s+",
		strings.Repeat("-", col1Width+2),
		strings.Repeat("-", col2Width+2),
		strings.Repeat("-", col3Width+2),
		strings.Repeat("-", col4Width+2),
		strings.Repeat("-", col5Width+2),
	))

	// Headers
	tableLines = append(tableLines, fmt.Sprintf("| %s | %s | %s | %s | %s |",
		padRight("服务名称", col1Width),
		padRight("配置端口", col2Width),
		padRight("配置来源 (YAML)", col3Width),
		padRight("当前状态", col4Width),
		padRight("占用进程", col5Width),
	))

	// Separator
	tableLines = append(tableLines, fmt.Sprintf("+%s+%s+%s+%s+%s+",
		strings.Repeat("-", col1Width+2),
		strings.Repeat("-", col2Width+2),
		strings.Repeat("-", col3Width+2),
		strings.Repeat("-", col4Width+2),
		strings.Repeat("-", col5Width+2),
	))

	// Row data
	for _, r := range rows {
		tableLines = append(tableLines, fmt.Sprintf("| %s | %s | %s | %s | %s |",
			padRight(r.Service, col1Width),
			padRight(strconv.Itoa(r.ConfigPort), col2Width),
			padRight(r.ConfigKey, col3Width),
			padRight(r.Status, col4Width),
			padRight(r.Occupant, col5Width),
		))
	}

	// Bottom border
	tableLines = append(tableLines, fmt.Sprintf("+%s+%s+%s+%s+%s+",
		strings.Repeat("-", col1Width+2),
		strings.Repeat("-", col2Width+2),
		strings.Repeat("-", col3Width+2),
		strings.Repeat("-", col4Width+2),
		strings.Repeat("-", col5Width+2),
	))

	tableStr := "\n" + strings.Join(tableLines, "\n")

	if hasOccupied {
		item.Status = StatusFail
		item.Message = fmt.Sprintf("以下关键端口已被占用: %s%s", strings.Join(occupiedNames, ", "), tableStr)
		item.Suggestion = "请停止占用这些端口的本地服务，或者修改 configs/default.yaml 中对应组件的端口绑定配置。"
	} else {
		item.Status = StatusOk
		item.Message = "关键端口未被占用" + tableStr
	}
	return item
}

func checkCompilationTools() DiagnosticItem {
	item := DiagnosticItem{Name: "Nginx 编译依赖环境"}
	var missingCmds []string
	for _, cmd := range []string{"gcc", "make"} {
		if _, err := exec.LookPath(cmd); err != nil {
			missingCmds = append(missingCmds, cmd)
		}
	}

	// CentOS header check
	var missingLibs []string
	if sysutil.IsLinux() {
		if _, err := exec.LookPath("rpm"); err == nil {
			libs := []string{"pcre-devel", "zlib-devel", "openssl-devel"}
			for _, lib := range libs {
				cmd := exec.Command("rpm", "-q", lib)
				if err := cmd.Run(); err != nil {
					missingLibs = append(missingLibs, lib)
				}
			}
		}
	}

	var reasons []string
	if len(missingCmds) > 0 {
		reasons = append(reasons, fmt.Sprintf("缺少编译命令 %s", strings.Join(missingCmds, "/")))
	}
	if len(missingLibs) > 0 {
		reasons = append(reasons, fmt.Sprintf("缺少开发包 %s", strings.Join(missingLibs, ", ")))
	}

	if len(reasons) > 0 {
		item.Status = StatusWarn
		item.Message = strings.Join(reasons, "; ")
		item.Suggestion = "请在 CentOS 系统下运行 'yum install -y gcc make pcre-devel zlib-devel openssl-devel' 以安装依赖。"
	} else {
		item.Status = StatusOk
		item.Message = "Nginx 编译工具及开发包环境检查通过"
	}
	return item
}

func checkFileLimits() DiagnosticItem {
	item := DiagnosticItem{Name: "系统文件句柄限制"}

	var rlimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit)
	if err != nil {
		item.Status = StatusWarn
		item.Message = fmt.Sprintf("无法获取系统文件句柄限制: %v", err)
		return item
	}

	// Check current soft limit
	softLimit := rlimit.Cur
	if softLimit < 65535 {
		item.Status = StatusWarn
		item.Message = fmt.Sprintf("当前最大文件描述符限制较小 (ulimit -n = %d)", softLimit)
		item.Suggestion = "建议在 /etc/security/limits.conf 中配置 '* soft nofile 1000000' 和 '* hard nofile 1000000' 以优化性能。"
	} else {
		item.Status = StatusOk
		item.Message = fmt.Sprintf("文件描述符限制检查通过 (ulimit -n = %d)", softLimit)
	}

	return item
}

func visualWidth(s string) int {
	w := 0
	for _, r := range s {
		if r > 127 {
			w += 2
		} else {
			w += 1
		}
	}
	return w
}

func padRight(s string, target int) string {
	w := visualWidth(s)
	if w >= target {
		return s
	}
	return s + strings.Repeat(" ", target-w)
}
