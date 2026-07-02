package binary

import (
	"os/exec"
	"time"

	"OpsVault/internal/driver"

	"github.com/spf13/viper"
)

type BaseDriver struct {
	Name   string
	Config *viper.Viper
}

func NewBaseDriver(name string, cfg *viper.Viper) *BaseDriver {
	return &BaseDriver{Name: name, Config: cfg}
}

func (d *BaseDriver) Status() (*driver.ServiceStatus, error) {
	_, err := exec.LookPath(d.Name)
	status := &driver.ServiceStatus{
		Name:      d.Name,
		Mode:      driver.ModeBinary,
		UpdatedAt: time.Now(),
		Details:   map[string]string{},
	}
	if err != nil {
		status.Status = "not installed"
		return status, nil
	}
	status.Status = "installed"
	return status, nil
}
