package tui

func DashboardView() string {
	return "Services overview\n\n- Nginx: binary mode\n- MySQL/Redis/RocketMQ/RabbitMQ/Postgres: docker mode\n\nUse ←/→ to switch panels, q to quit."
}
