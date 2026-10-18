package config

import (
	"os"
	"strings"
	"testing"
)

func configure(t *testing.T) {
	t.Helper()
	t.Setenv("HEPH_DATABASE_URL", "postgresql://explicit:test-only-secret@127.0.0.1:15432/config_test?sslmode=disable")
	t.Setenv("HEPH_DATABASE_SCHEMA", "explicit_schema")
	t.Setenv("HEPH_LISTEN_ADDR", "127.0.0.1:18080")
	t.Setenv("HEPH_BOOTSTRAP_IDENTITIES", `[{"source":"test","subject":"admin","token":"unit-test-credential-placeholder"}]`)
}

func TestFromEnv(t *testing.T) {
	configure(t)
	cfg, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddr.String() != "127.0.0.1:18080" || cfg.DatabaseSchema != "explicit_schema" || cfg.DatabaseURL != "postgresql://explicit:test-only-secret@127.0.0.1:15432/config_test?sslmode=disable" || len(cfg.Identities) != 1 {
		t.Fatalf("configuration did not preserve explicit settings")
	}
}

func TestRequiredEnvironment(t *testing.T) {
	for _, name := range []string{"HEPH_LISTEN_ADDR", "HEPH_DATABASE_URL", "HEPH_DATABASE_SCHEMA", "HEPH_BOOTSTRAP_IDENTITIES"} {
		for _, value := range []string{"unset", "", " \t\n"} {
			t.Run(name+"/"+value, func(t *testing.T) {
				configure(t)
				t.Setenv(name, value)
				if value == "unset" {
					if err := os.Unsetenv(name); err != nil {
						t.Fatal(err)
					}
				}
				_, err := FromEnv()
				if err == nil || !strings.Contains(err.Error(), name+" must be set") {
					t.Fatalf("missing %s must fail clearly, got %v", name, err)
				}
				if strings.Contains(err.Error(), "test-only-secret") || strings.Contains(err.Error(), "unit-test-credential") {
					t.Fatal("configuration error exposed a credential")
				}
			})
		}
	}
}

func TestInvalidServerSettings(t *testing.T) {
	for _, tc := range []struct{ name, value string }{
		{"HEPH_LISTEN_ADDR", "127.0.0.1"},
		{"HEPH_LISTEN_ADDR", "127.0.0.1:70000"},
		{"HEPH_BOOTSTRAP_IDENTITIES", "secret-malformed-json"},
		{"HEPH_BOOTSTRAP_IDENTITIES", "[]"},
		{"HEPH_BOOTSTRAP_IDENTITIES", "null"},
		{"HEPH_BOOTSTRAP_IDENTITIES", `[{"source":"test","subject":"admin","token":"secret-too-short"}]`},
	} {
		t.Run(tc.name+"/"+tc.value, func(t *testing.T) {
			configure(t)
			t.Setenv(tc.name, tc.value)
			_, err := FromEnv()
			if err == nil || !strings.Contains(err.Error(), tc.name) || strings.Contains(err.Error(), "secret-") {
				t.Fatalf("invalid setting must fail without its contents, got %v", err)
			}
		})
	}
	t.Run("ipv6", func(t *testing.T) {
		configure(t)
		t.Setenv("HEPH_LISTEN_ADDR", "[::1]:18080")
		cfg, err := FromEnv()
		if err != nil || cfg.ListenAddr.String() != "[::1]:18080" {
			t.Fatalf("IPv6 setting rejected: %v", err)
		}
	})
}

func TestDatabaseCommandsNeedOnlyDatabaseSettings(t *testing.T) {
	configure(t)
	t.Setenv("HEPH_LISTEN_ADDR", "invalid server-only setting")
	t.Setenv("HEPH_BOOTSTRAP_IDENTITIES", "invalid server-only setting")
	cfg, err := DatabaseFromEnv()
	if err != nil || cfg.Schema != "explicit_schema" {
		t.Fatalf("database configuration must not depend on server settings: %v", err)
	}
}
