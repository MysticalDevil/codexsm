package util

import "testing"

func TestParseOlderThan(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"30d", "720h0m0s"},
		{"12h", "12h0m0s"},
		{"45m", "45m0s"},
		{"30s", "30s"},
		{"1h30m", "1h30m0s"},
	}
	for _, tc := range tests {
		got, err := ParseOlderThan(tc.in)
		if err != nil {
			t.Fatalf("ParseOlderThan(%q): %v", tc.in, err)
		}
		if got.String() != tc.want {
			t.Fatalf("ParseOlderThan(%q)=%s want %s", tc.in, got.String(), tc.want)
		}
	}
}

func TestParseOlderThanInvalid(t *testing.T) {
	if _, err := ParseOlderThan("abc"); err == nil {
		t.Fatal("expected error for invalid duration")
	}
}
