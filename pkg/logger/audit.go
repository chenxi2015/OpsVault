package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	auditLogger *log.Logger
	auditOnce   sync.Once
	auditPath   string
)

// ConfigureAudit initializes the audit logger to write to the given log directory.
// It is safe to call multiple times; only the first call takes effect.
func ConfigureAudit(logDir string) {
	auditOnce.Do(func() {
		if err := os.MkdirAll(logDir, 0o755); err != nil {
			// Fall back to stderr if we cannot create the log directory
			auditLogger = log.New(os.Stderr, "[AUDIT] ", log.LstdFlags)
			return
		}
		auditPath = filepath.Join(logDir, "audit.log")
		f, err := os.OpenFile(auditPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			auditLogger = log.New(os.Stderr, "[AUDIT] ", log.LstdFlags)
			return
		}
		auditLogger = log.New(f, "", 0)
	})
}

// AuditLog records a structured audit entry for destructive/important operations.
// service: affected service (e.g. "mysql"), action: operation name (e.g. "install"),
// detail: any extra info (e.g. "version=8.0"), success: whether the operation succeeded.
func AuditLog(service, action, detail string, success bool) {
	if auditLogger == nil {
		// Lazy init to stderr when not explicitly configured
		auditLogger = log.New(os.Stderr, "", 0)
	}
	result := "SUCCESS"
	if !success {
		result = "FAILED"
	}
	uid := os.Getuid()
	entry := fmt.Sprintf("%s [AUDIT] uid=%d service=%s action=%s result=%s %s",
		time.Now().Format(time.RFC3339),
		uid,
		service,
		action,
		result,
		detail,
	)
	auditLogger.Println(entry)
}
