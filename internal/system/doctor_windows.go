//go:build windows

package system

// checkFileLimitsPosix is not available on Windows (no rlimit concept).
// This stub satisfies the compiler when cross-compiling for Windows.
func checkFileLimitsPosix() DiagnosticItem {
	return DiagnosticItem{
		Name:    "系统文件句柄限制",
		Status:  StatusOk,
		Message: "Windows 平台不适用文件句柄限制检查 (无 ulimit 概念)",
	}
}
