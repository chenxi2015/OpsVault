package backup

import "time"

// BackupMetadata holds information about a configuration backup.
type BackupMetadata struct {
	Name        string    `json:"name"`
	Timestamp   time.Time `json:"timestamp"`
	Services    []string  `json:"services"`
	SizeBytes   int64     `json:"size_bytes"`
	Description string    `json:"description"`
}
