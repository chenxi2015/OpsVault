package mysql

import "github.com/docker/docker/api/types/container"

func containerLogsOptions() container.LogsOptions {
	return container.LogsOptions{ShowStdout: true, ShowStderr: true, Tail: "100"}
}
