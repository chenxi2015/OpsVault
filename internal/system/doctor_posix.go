//go:build !windows

package system

import (
	"fmt"
	"syscall"
)

// checkFileLimitsPosix reads the process file descriptor rlimit on POSIX systems.
func checkFileLimitsPosix() DiagnosticItem {
	item := DiagnosticItem{Name: "系统文件句柄限制"}
	var rlimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit)
	if err != nil {
		item.Status = StatusWarn
		item.Message = fmt.Sprintf("无法获取系统文件句柄限制: %v", err)
		return item
	}
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
