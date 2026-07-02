package system

import "os/exec"

func StartService(name string) error {
	return exec.Command("systemctl", "start", name).Run()
}

func StopService(name string) error {
	return exec.Command("systemctl", "stop", name).Run()
}

func RestartService(name string) error {
	return exec.Command("systemctl", "restart", name).Run()
}

func EnableService(name string) error {
	return exec.Command("systemctl", "enable", name).Run()
}
