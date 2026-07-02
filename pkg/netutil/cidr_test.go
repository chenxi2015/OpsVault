package netutil

import "testing"

func TestValidateCIDR(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		wantErr bool
	}{
		{name: "valid", cidr: "172.28.0.0/16"},
		{name: "missing mask", cidr: "172.28.0.0", wantErr: true},
		{name: "invalid ip", cidr: "999.28.0.0/16", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCIDR(tt.cidr)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for %q", tt.cidr)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.cidr, err)
			}
		})
	}
}
