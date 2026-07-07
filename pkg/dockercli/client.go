package dockercli

import (
	"context"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/spf13/viper"
)

func ResolveNetworkName(cfg *viper.Viper) string {
	if cfg == nil {
		return "opsvault-net"
	}
	prefix := cfg.GetString("docker.name_prefix")
	if prefix == "" {
		prefix = "opsvault"
	}
	netName := cfg.GetString("docker.network_name")
	if netName == "" || netName == "opsvault-net" {
		if prefix != "opsvault" {
			return prefix + "-net"
		}
	}
	if netName == "" {
		return prefix + "-net"
	}
	return netName
}

func ResolveContainerName(cfg *viper.Viper, name string) string {
	if cfg == nil {
		return "opsvault-" + name
	}
	prefix := cfg.GetString("docker.name_prefix")
	if prefix == "" {
		prefix = "opsvault"
	}
	return prefix + "-" + name
}


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
