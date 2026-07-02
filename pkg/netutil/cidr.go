package netutil

import (
	"fmt"
	"net"
)

func ValidateCIDR(cidr string) error {
	if cidr == "" {
		return fmt.Errorf("cidr is required")
	}
	if _, _, err := net.ParseCIDR(cidr); err != nil {
		return fmt.Errorf("invalid cidr %q: %w", cidr, err)
	}
	return nil
}
