package tui

import (
	"fmt"
	"time"

	"OpsVault/internal/driver"
	dockdrv "OpsVault/internal/driver/docker"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/client"
	"github.com/spf13/viper"
)

type taskFinishedMsg struct {
	ServiceName string
	ActionName  string
	Output      string
	Err         error
}

func runAction(cfg *viper.Viper, dockerCli *client.Client, service ServiceRef, action Action, params map[string]string) tea.Cmd {
	return func() tea.Msg {
		var drv driver.ServiceDriver
		drv = service.Driver
		name := service.Name

		// Dynamically reconstruct driver with passwords/options for installation if needed
		if action.ID == ActionInstall {
			switch name {
			case "mysql":
				rootPwd := params["root-pwd"]
				if rootPwd == "" {
					rootPwd = "root"
				}
				drv = dockdrv.NewMySQLDriver(dockdrv.WrapClient(dockerCli), cfg, rootPwd)
			case "redis":
				pwd := params["pwd"]
				drv = dockdrv.NewRedisDriver(dockdrv.WrapClient(dockerCli), cfg, pwd)
			case "rabbitmq":
				user := params["admin-user"]
				if user == "" {
					user = "admin"
				}
				pass := params["admin-pwd"]
				if pass == "" {
					pass = "123456"
				}
				drv = dockdrv.NewRabbitMQDriver(dockdrv.WrapClient(dockerCli), cfg, user, pass)
			case "postgres":
				pwd := params["pwd"]
				if pwd == "" {
					pwd = "postgres"
				}
				drv = dockdrv.NewPostgresDriver(dockdrv.WrapClient(dockerCli), cfg, pwd)
			}
		}

		var err error
		var output string

		switch action.ID {
		case ActionInstall:
			err = drv.Install()
			output = fmt.Sprintf("Installation completed for service %s.", name)
		case ActionStart:
			err = drv.Start()
			output = fmt.Sprintf("Started service %s.", name)
		case ActionStop:
			err = drv.Stop()
			output = fmt.Sprintf("Stopped service %s.", name)
		case ActionRestart:
			err = drv.Restart()
			output = fmt.Sprintf("Restarted service %s.", name)
		case ActionUpgrade:
			targetVersion := params["version"]
			if targetVersion == "" {
				switch name {
				case "mysql":
					targetVersion = "8.4"
				case "redis":
					targetVersion = "7.2-alpine"
				case "rocketmq":
					targetVersion = "5.3.0"
				case "rabbitmq":
					targetVersion = "3.13-management"
				default:
					targetVersion = "latest"
				}
			}
			err = drv.Upgrade(targetVersion)
			output = fmt.Sprintf("Upgraded service %s to version %s.", name, targetVersion)
		case ActionUninstall:
			purge := params["purge"] == "true"
			err = drv.Uninstall(purge)
			output = fmt.Sprintf("Uninstalled service %s (purge data: %v).", name, purge)
		case ActionReload:
			if reloadable, ok := drv.(driver.Reloadable); ok {
				err = reloadable.Reload()
				output = fmt.Sprintf("Reloaded configuration for service %s.", name)
			} else {
				err = fmt.Errorf("service %s does not support reload", name)
			}
		case ActionLogs:
			if reader, ok := drv.(driver.LogReader); ok {
				var logData string
				logData, err = reader.TailLogs(100)
				if err == nil {
					output = logData
				}
			} else {
				err = fmt.Errorf("service %s does not support log reading", name)
			}
		case "vhost_add":
			if mgr, ok := drv.(driver.VHostManager); ok {
				err = mgr.AddVHost(params["domain"], params["root"])
				output = fmt.Sprintf("Successfully added virtual host for %s.", params["domain"])
			} else {
				err = fmt.Errorf("service %s does not support virtual host management", name)
			}
		case "vhost_del":
			if mgr, ok := drv.(driver.VHostManager); ok {
				purge := params["delete-root"] == "true"
				err = mgr.DeleteVHost(params["domain"], purge)
				output = fmt.Sprintf("Successfully deleted virtual host for %s.", params["domain"])
			} else {
				err = fmt.Errorf("service %s does not support virtual host management", name)
			}
		case "ssl_apply":
			if mgr, ok := drv.(driver.SSLManager); ok {
				err = mgr.ApplySSL(params["domain"])
				output = fmt.Sprintf("Successfully applied Let's Encrypt SSL certificate for %s.", params["domain"])
			} else {
				err = fmt.Errorf("service %s does not support SSL management", name)
			}
		case "ssl_renew":
			if mgr, ok := drv.(driver.SSLManager); ok {
				err = mgr.RenewSSL(params["domain"])
				output = fmt.Sprintf("Successfully renewed Let's Encrypt SSL certificate for %s.", params["domain"])
			} else {
				err = fmt.Errorf("service %s does not support SSL management", name)
			}
		case "ssl_delete":
			if mgr, ok := drv.(driver.SSLManager); ok {
				err = mgr.DeleteSSL(params["domain"])
				output = fmt.Sprintf("Successfully deleted SSL certificate for %s.", params["domain"])
			} else {
				err = fmt.Errorf("service %s does not support SSL management", name)
			}
		default:
			err = fmt.Errorf("unknown action: %s", action.ID)
		}

		if err != nil {
			output = fmt.Sprintf("Error running %s on %s:\n%v", action.ID, name, err)
		}

		// Artificial small delay to make the task transition visible in UI
		time.Sleep(200 * time.Millisecond)

		return taskFinishedMsg{
			ServiceName: name,
			ActionName:  string(action.ID),
			Output:      output,
			Err:         err,
		}
	}
}
