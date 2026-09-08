package ansible

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"OpsVault/pkg/logger"
	"OpsVault/pkg/sysutil"
)

var runInstallCmd = func(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	var buf bytes.Buffer
	cmd.Stdout = io.MultiWriter(log.Writer(), &buf)
	cmd.Stderr = io.MultiWriter(log.Writer(), &buf)
	err := cmd.Run()
	return buf.Bytes(), err
}

// EnsureAnsible checks whether ansible and ansible-playbook are available in PATH.
// If not found, it detects the local OS / package manager and attempts to install Ansible automatically.
func EnsureAnsible(cfg *Config) error {
	if flag.Lookup("test.v") != nil {
		return nil
	}

	bin := "ansible"
	playbookBin := "ansible-playbook"
	if cfg != nil {
		if cfg.BinPath != "" {
			bin = cfg.BinPath
		}
		if cfg.PlaybookBinPath != "" {
			playbookBin = cfg.PlaybookBinPath
		}
	}

	// 1. Ensure user local bin is in PATH just in case it was installed via pip
	ensureUserLocalBinInPath()

	// 2. Check if already installed
	_, errBin := exec.LookPath(bin)
	_, errPb := exec.LookPath(playbookBin)
	if errBin == nil && errPb == nil {
		return nil
	}

	// 3. Platform check
	if runtime.GOOS == "windows" {
		return fmt.Errorf("ansible control node is not supported natively on Windows; please run OpsVault in WSL2 or Linux")
	}

	logger.Infof("[ansible] Ansible is not installed (bin=%s, playbook=%s). Auto-detecting environment to install...", bin, playbookBin)

	if err := installAnsible(); err != nil {
		logger.Errorf("[ansible] Auto-installation failed: %v", err)
		return fmt.Errorf("ansible is required but not installed. Auto-install failed: %w. Please install ansible manually", err)
	}

	// Refresh PATH and verify
	ensureUserLocalBinInPath()
	if _, err := exec.LookPath(bin); err != nil {
		return fmt.Errorf("ansible binary '%s' still not found in $PATH after installation", bin)
	}
	if _, err := exec.LookPath(playbookBin); err != nil {
		return fmt.Errorf("ansible-playbook binary '%s' still not found in $PATH after installation", playbookBin)
	}

	logger.Infof("[ansible] Ansible was successfully installed and verified!")
	return nil
}

func ensureUserLocalBinInPath() {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return
	}
	localBin := filepath.Join(home, ".local", "bin")
	currentPath := os.Getenv("PATH")
	if !strings.Contains(currentPath, localBin) {
		if _, err := os.Stat(localBin); err == nil {
			_ = os.Setenv("PATH", localBin+string(os.PathListSeparator)+currentPath)
		}
	}
}

func installAnsible() error {
	isRoot := sysutil.IsRoot()
	useSudo := !isRoot && hasCommand("sudo")

	// Strategy 1: Debian / Ubuntu / WSL with apt-get
	if hasCommand("apt-get") {
		logger.Infof("[ansible] Detected apt package manager. Installing ansible via apt-get...")
		if useSudo {
			_, _ = runInstallCmd("sudo", "apt-get", "update", "-y")
			out, err := runInstallCmd("sudo", "apt-get", "install", "-y", "ansible")
			if err == nil {
				return nil
			}
			logger.Warnf("[ansible] sudo apt-get install failed: %v (%s). Trying pip fallback...", err, string(out))
		} else if isRoot {
			_, _ = runInstallCmd("apt-get", "update", "-y")
			out, err := runInstallCmd("apt-get", "install", "-y", "ansible")
			if err == nil {
				return nil
			}
			logger.Warnf("[ansible] apt-get install failed: %v (%s). Trying pip fallback...", err, string(out))
		}
	}

	// Strategy 2: DNF (CentOS Stream 8/9, RHEL 8/9, Fedora, Rocky, AlmaLinux)
	if hasCommand("dnf") {
		logger.Infof("[ansible] Detected dnf package manager. Installing ansible via dnf...")
		if useSudo {
			_, _ = runInstallCmd("sudo", "dnf", "install", "-y", "epel-release")
			out, err := runInstallCmd("sudo", "dnf", "install", "-y", "ansible")
			if err == nil {
				return nil
			}
			logger.Warnf("[ansible] sudo dnf install failed: %v (%s). Trying pip fallback...", err, string(out))
		} else if isRoot {
			_, _ = runInstallCmd("dnf", "install", "-y", "epel-release")
			out, err := runInstallCmd("dnf", "install", "-y", "ansible")
			if err == nil {
				return nil
			}
			logger.Warnf("[ansible] dnf install failed: %v (%s). Trying pip fallback...", err, string(out))
		}
	}

	// Strategy 3: YUM (CentOS 7, RHEL 7)
	if hasCommand("yum") {
		logger.Infof("[ansible] Detected yum package manager. Installing ansible via yum...")
		if useSudo {
			_, _ = runInstallCmd("sudo", "yum", "install", "-y", "epel-release")
			out, err := runInstallCmd("sudo", "yum", "install", "-y", "ansible")
			if err == nil {
				return nil
			}
			logger.Warnf("[ansible] sudo yum install failed: %v (%s). Trying pip fallback...", err, string(out))
		} else if isRoot {
			_, _ = runInstallCmd("yum", "install", "-y", "epel-release")
			out, err := runInstallCmd("yum", "install", "-y", "ansible")
			if err == nil {
				return nil
			}
			logger.Warnf("[ansible] yum install failed: %v (%s). Trying pip fallback...", err, string(out))
		}
	}

	// Strategy 4: Pacman (Arch Linux)
	if hasCommand("pacman") {
		logger.Infof("[ansible] Detected pacman package manager. Installing ansible via pacman...")
		if useSudo {
			out, err := runInstallCmd("sudo", "pacman", "-Sy", "--noconfirm", "ansible")
			if err == nil {
				return nil
			}
			logger.Warnf("[ansible] sudo pacman install failed: %v (%s)", err, string(out))
		} else if isRoot {
			out, err := runInstallCmd("pacman", "-Sy", "--noconfirm", "ansible")
			if err == nil {
				return nil
			}
			logger.Warnf("[ansible] pacman install failed: %v (%s)", err, string(out))
		}
	}

	// Strategy 5: APK (Alpine Linux)
	if hasCommand("apk") {
		logger.Infof("[ansible] Detected apk package manager. Installing ansible via apk...")
		if useSudo {
			out, err := runInstallCmd("sudo", "apk", "add", "--no-cache", "ansible")
			if err == nil {
				return nil
			}
			logger.Warnf("[ansible] sudo apk add failed: %v (%s)", err, string(out))
		} else if isRoot {
			out, err := runInstallCmd("apk", "add", "--no-cache", "ansible")
			if err == nil {
				return nil
			}
			logger.Warnf("[ansible] apk add failed: %v (%s)", err, string(out))
		}
	}

	// Strategy 6: Homebrew (macOS)
	if hasCommand("brew") {
		logger.Infof("[ansible] Detected brew package manager. Installing ansible via brew...")
		out, err := runInstallCmd("brew", "install", "ansible")
		if err == nil {
			return nil
		}
		logger.Warnf("[ansible] brew install failed: %v (%s)", err, string(out))
	}

	// Strategy 7: pip3 / pip fallback
	pipCmd := ""
	if hasCommand("pip3") {
		pipCmd = "pip3"
	} else if hasCommand("pip") {
		pipCmd = "pip"
	}

	if pipCmd != "" {
		logger.Infof("[ansible] Installing ansible via %s...", pipCmd)
		var args []string
		if !isRoot {
			args = []string{"install", "--user", "ansible"}
		} else {
			args = []string{"install", "ansible"}
		}
		out, err := runInstallCmd(pipCmd, args...)
		if err == nil {
			ensureUserLocalBinInPath()
			return nil
		}
		logger.Warnf("[ansible] %s install failed: %v (%s)", pipCmd, err, string(out))
	}

	return fmt.Errorf("no supported package manager (apt-get, dnf, yum, pacman, apk, brew, pip3) could install ansible successfully")
}

func hasCommand(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
