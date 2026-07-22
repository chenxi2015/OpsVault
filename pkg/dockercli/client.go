package dockercli

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"

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

func CleanOrphanedBridges(ctx context.Context, cli *client.Client) error {
	if cli == nil {
		return nil
	}
	// /sys/class/net and `ip link` are Linux-only; skip on other platforms.
	if runtime.GOOS != "linux" {
		return nil
	}
	networks, err := cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return err
	}
	activeIds := make(map[string]bool)
	for _, nw := range networks {
		if len(nw.ID) >= 12 {
			activeIds[nw.ID[:12]] = true
		} else {
			activeIds[nw.ID] = true
		}
	}

	files, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return nil
	}

	for _, f := range files {
		name := f.Name()
		if strings.HasPrefix(name, "br-") {
			idPart := strings.TrimPrefix(name, "br-")
			if len(idPart) == 12 {
				if !activeIds[idPart] {
					_ = exec.CommandContext(ctx, "ip", "link", "delete", name).Run()
				}
			}
		}
	}
	return nil
}

func EnsureNetwork(ctx context.Context, cli *client.Client, name, cidr string) error {
	if cli == nil {
		return nil
	}
	_ = CleanOrphanedBridges(ctx, cli)
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
