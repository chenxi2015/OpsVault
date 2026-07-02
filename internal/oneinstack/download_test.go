package oneinstack

import "testing"

func TestScriptFileNameFromURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "default auto url",
			url:  "https://oneinstack.com/auto/",
			want: "oneinstack-auto.sh",
		},
		{
			name: "explicit file",
			url:  "https://example.com/scripts/install.sh",
			want: "install.sh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scriptFileNameFromURL(tt.url)
			if got != tt.want {
				t.Fatalf("scriptFileNameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
