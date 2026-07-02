package dockercli

import (
	"context"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

func New() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

func EnsureNetwork(ctx context.Context, cli *client.Client, name, cidr string) error {
	if cli == nil {
		return nil
	}
	existing, err := cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return err
	}
	for _, nw := range existing {
		if nw.Name == name {
			return nil
		}
	}
	_, err = cli.NetworkCreate(ctx, name, network.CreateOptions{
		Driver: "bridge",
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{{Subnet: cidr}},
		},
	})
	return err
}
