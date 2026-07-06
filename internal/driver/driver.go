package driver

import "time"

type Mode string

const (
	ModeDocker Mode = "docker"
	ModeBinary Mode = "binary"
)

type ServiceStatus struct {
	Name         string
	Mode         Mode
	Running      bool
	Status       string
	Version      string
	Ports        []string
	DataPath     string
	Network      string
	StorageUsage string
	PID          int
	UpdatedAt    time.Time
	Details      map[string]string
}

type ServiceDriver interface {
	Install() error
	Start() error
	Stop() error
	Restart() error
	Uninstall(purgeData bool) error
	Upgrade(targetVersion string) error
	Status() (*ServiceStatus, error)
}

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
