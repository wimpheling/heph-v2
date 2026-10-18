package config

import (
	"os"
	"testing"
)

func TestFromEnv(t *testing.T) {
	t.Setenv("HEPH_BOOTSTRAP_IDENTITIES", `[{"source":"test","subject":"admin","token":"unit-test-credential-placeholder"}]`)
	for _, tc := range []struct {
		name    string
		value   string
		absent  bool
		want    string
		invalid bool
	}{
		{name: "default", absent: true, want: "127.0.0.1:8080"},
		{name: "ipv4", value: "127.0.0.1:18080", want: "127.0.0.1:18080"},
		{name: "ipv6", value: "[::1]:8080", want: "[::1]:8080"},
		{name: "empty", invalid: true},
		{name: "missing port", value: "127.0.0.1", invalid: true},
		{name: "invalid port", value: "127.0.0.1:70000", invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HEPH_LISTEN_ADDR", tc.value)
			if tc.absent {
				if err := os.Unsetenv("HEPH_LISTEN_ADDR"); err != nil {
					t.Fatal(err)
				}
			}
			cfg, err := FromEnv()
			if tc.invalid {
				if err == nil {
					t.Fatal("expected invalid configuration error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := cfg.ListenAddr.String(); got != tc.want {
				t.Fatalf("address = %q, want %q", got, tc.want)
			}
		})
	}
}
