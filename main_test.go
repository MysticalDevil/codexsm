package main

import "testing"

func TestNormalizeBuildInfoVersion(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: ""},
		{in: "   ", want: ""},
		{in: "(devel)", want: ""},
		{in: "v0.1.4", want: "v0.1.4"},
		{in: "  v1.2.3  ", want: "v1.2.3"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.in, func(t *testing.T) {
			if got := normalizeBuildInfoVersion(tt.in); got != tt.want {
				t.Fatalf("normalizeBuildInfoVersion(%q)=%q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

