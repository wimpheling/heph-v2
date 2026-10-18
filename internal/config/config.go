// Package config loads and validates process configuration at startup.
package config

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"os"

	bootstrap "github.com/wimpheling/heph-v2/internal/drivers/bootstrap/identity"
)

type Config struct {
	ListenAddr     netip.AddrPort
	DatabaseURL    string
	DatabaseSchema string
	Identities     []bootstrap.Credential
}

func FromEnv() (Config, error) {
	address, present := os.LookupEnv("HEPH_LISTEN_ADDR")
	if !present {
		address = "127.0.0.1:8080"
	}
	parsed, err := netip.ParseAddrPort(address)
	if err != nil {
		return Config{}, fmt.Errorf("HEPH_LISTEN_ADDR: %w", err)
	}
	url := os.Getenv("HEPH_DATABASE_URL")
	if url == "" {
		url = "postgresql://heph:heph-local-placeholder@127.0.0.1:5432/heph?sslmode=disable"
	}
	schema := os.Getenv("HEPH_DATABASE_SCHEMA")
	if schema == "" {
		schema = "public"
	}
	var identities []bootstrap.Credential
	raw := os.Getenv("HEPH_BOOTSTRAP_IDENTITIES")
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &identities); err != nil {
			return Config{}, fmt.Errorf("HEPH_BOOTSTRAP_IDENTITIES must contain a JSON array of configured identities")
		}
		if _, err := bootstrap.New(identities); err != nil {
			return Config{}, err
		}
	}
	return Config{ListenAddr: parsed, DatabaseURL: url, DatabaseSchema: schema, Identities: identities}, nil
}
