// Package config loads and validates process configuration at startup.
package config

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"strings"

	bootstrap "github.com/wimpheling/heph-v2/internal/drivers/bootstrap/identity"
)

type Config struct {
	ListenAddr     netip.AddrPort
	DatabaseURL    string
	DatabaseSchema string
	Identities     []bootstrap.Credential
}

func FromEnv() (Config, error) {
	database, err := DatabaseFromEnv()
	if err != nil {
		return Config{}, err
	}
	address, err := requiredEnv("HEPH_LISTEN_ADDR")
	if err != nil {
		return Config{}, err
	}
	parsed, err := netip.ParseAddrPort(address)
	if err != nil {
		return Config{}, fmt.Errorf("HEPH_LISTEN_ADDR must contain an IP address and port")
	}
	raw, err := requiredEnv("HEPH_BOOTSTRAP_IDENTITIES")
	if err != nil {
		return Config{}, err
	}
	var identities []bootstrap.Credential
	if err := json.Unmarshal([]byte(raw), &identities); err != nil {
		return Config{}, fmt.Errorf("HEPH_BOOTSTRAP_IDENTITIES must contain a JSON array of configured identities")
	}
	if _, err := bootstrap.New(identities); err != nil {
		return Config{}, fmt.Errorf("HEPH_BOOTSTRAP_IDENTITIES: %w", err)
	}
	return Config{ListenAddr: parsed, DatabaseURL: database.URL, DatabaseSchema: database.Schema, Identities: identities}, nil
}

type Database struct {
	URL    string
	Schema string
}

// DatabaseFromEnv loads only the settings used by migrate and reset.
func DatabaseFromEnv() (Database, error) {
	url, err := requiredEnv("HEPH_DATABASE_URL")
	if err != nil {
		return Database{}, err
	}
	schema, err := requiredEnv("HEPH_DATABASE_SCHEMA")
	if err != nil {
		return Database{}, err
	}
	return Database{URL: url, Schema: schema}, nil
}

// Configuration has no hardcoded environment defaults. Values must be explicit.
func requiredEnv(name string) (string, error) {
	value, present := os.LookupEnv(name)
	if !present || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s must be set to a non-empty value", name)
	}
	return value, nil
}
