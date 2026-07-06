package tui

import (
	"OpsVault/internal/driver"
)

type ActionID string

const (
	ActionInstall   ActionID = "install"
	ActionStart     ActionID = "start"
	ActionStop      ActionID = "stop"
	ActionRestart   ActionID = "restart"
	ActionUpgrade   ActionID = "upgrade"
	ActionUninstall ActionID = "uninstall"
	ActionLogs      ActionID = "logs"
	ActionReload    ActionID = "reload"
	ActionVersion   ActionID = "version"
	ActionDLQStat   ActionID = "dlq"
)

type Action struct {
	ID        ActionID
	Label     string
	Dangerous bool
}

type ServiceRef struct {
	Name   string
	Driver driver.ServiceDriver
}

func AvailableServiceActions(status driver.ServiceStatus, svc ServiceRef) []Action {
	if status.Status == "not installed" {
		return []Action{
			{ID: ActionInstall, Label: "Install", Dangerous: false},
		}
	}

	var actions []Action
	if status.Running {
		actions = append(actions, Action{ID: ActionStop, Label: "Stop", Dangerous: false})
		actions = append(actions, Action{ID: ActionRestart, Label: "Restart", Dangerous: false})
		if _, ok := svc.Driver.(driver.Reloadable); ok {
			actions = append(actions, Action{ID: ActionReload, Label: "Reload", Dangerous: false})
		}
	} else {
		actions = append(actions, Action{ID: ActionStart, Label: "Start", Dangerous: false})
	}

	actions = append(actions, Action{ID: ActionUpgrade, Label: "Upgrade", Dangerous: false})
	actions = append(actions, Action{ID: ActionUninstall, Label: "Uninstall", Dangerous: true})

	if _, ok := svc.Driver.(driver.LogReader); ok {
		actions = append(actions, Action{ID: ActionLogs, Label: "Logs", Dangerous: false})
	}

	return actions
}
