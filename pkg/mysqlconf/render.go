// Package mysqlconf provides shared MySQL configuration rendering.
// Used by both the local Docker driver and the Ansible deploy path.
package mysqlconf

// RenderMyCnf returns the content for /etc/mysql/conf.d/my.cnf.
func RenderMyCnf() string {
	return `[mysqld]
user=mysql
skip-name-resolve
default-storage-engine=InnoDB
character-set-server=utf8mb4
collation-server=utf8mb4_unicode_ci
max_connections=1000
innodb_buffer_pool_size=256M
innodb_log_file_size=64M
slow_query_log=1
slow_query_log_file=/var/log/mysql/slow.log
long_query_time=2
sql_mode=STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION

[client]
default-character-set=utf8mb4

[mysql]
default-character-set=utf8mb4
`
}
