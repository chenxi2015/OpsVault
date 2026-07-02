package docker

import (
	"context"

	"OpsVault/pkg/dockercli"
)

func (d *BaseDriver) EnsureNetwork(ctx context.Context) error {
	return dockercli.EnsureNetwork(ctx, d.Client, d.NetworkName, d.Config.GetString("docker.cidr"))
}
