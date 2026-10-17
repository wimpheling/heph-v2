// Package config loads and validates process configuration at startup.
package config

import (
	"fmt"
	"net/netip"
	"os"
)

type Config struct {
	ListenAddr netip.AddrPort
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
	return Config{ListenAddr: parsed}, nil
}
