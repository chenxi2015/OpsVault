// Package redisconf provides shared Redis configuration rendering.
// Used by both the local Docker driver and the Ansible deploy path.
package redisconf

import "fmt"

// RenderRedisCnf returns the content for redis.conf.
// password may be empty (no auth).
func RenderRedisCnf(password string) string {
	base := `bind 0.0.0.0
protected-mode no
port 6379
tcp-backlog 511
timeout 0
tcp-keepalive 300
daemonize no
supervised no
loglevel notice
databases 16
always-show-logo yes

# RDB persistence
save 900 1
save 300 10
save 60 10000
stop-writes-on-bgsave-error yes
rdbcompression yes
rdbchecksum yes
dbfilename dump.rdb
dir /data

# AOF persistence
appendonly yes
appendfilename "appendonly.aof"
appendfsync everysec
no-appendfsync-on-rewrite no
auto-aof-rewrite-percentage 100
auto-aof-rewrite-min-size 64mb
aof-load-truncated yes
aof-use-rdb-preamble yes
`
	if password != "" {
		base += fmt.Sprintf("\nrequirepass \"%s\"\n", password)
	}
	return base
}
