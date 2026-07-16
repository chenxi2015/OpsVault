// Package rabbitmqconf provides shared RabbitMQ configuration rendering.
// Used by both the local Docker driver and the Ansible deploy path.
package rabbitmqconf

import "fmt"

// RenderRabbitMQConf returns the content for /etc/rabbitmq/rabbitmq.conf.
func RenderRabbitMQConf(user, pass string) string {
	return fmt.Sprintf(`loopback_users.guest = false
listeners.tcp.default = 5672
management.listener.port = 15672
default_user = %s
default_pass = %s
`, user, pass)
}
