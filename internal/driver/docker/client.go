package docker

import "github.com/docker/docker/client"

type DockerClient interface {
	Raw() *client.Client
}

type ClientWrapper struct {
	client *client.Client
}

func WrapClient(cli *client.Client) ClientWrapper {
	return ClientWrapper{client: cli}
}

func (c ClientWrapper) Raw() *client.Client {
	return c.client
}
