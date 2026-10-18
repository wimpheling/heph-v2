package config

import (
	"errors"
	"os"
)

func assignment() string {
	value := os.Getenv("HEPH_ADDRESS") // want "environment values must not have fallback defaults"
	if value == "" {
		value = "127.0.0.1:8080"
	}
	return value
}

func lookup() string {
	value, present := os.LookupEnv("HEPH_ADDRESS") // want "environment values must not have fallback defaults"
	if !present {
		value = "127.0.0.1:8080"
	}
	return value
}

func earlyReturn() string {
	value, present := os.LookupEnv("HEPH_SCHEMA")
	if !present {
		return "public" // want "environment values must not have fallback defaults"
	}
	return value
}

func required() (string, error) {
	value, present := os.LookupEnv("HEPH_ADDRESS")
	if !present || value == "" {
		return "", errors.New("HEPH_ADDRESS is required")
	}
	return value, nil
}

func emptyFallback() (string, error) {
	value, present := os.LookupEnv("HEPH_ADDRESS")
	if !present {
		return "", nil // want "environment values must not have fallback defaults"
	}
	return value, nil
}

func wrapped() string {
	value, err := required() // want "environment values must not have fallback defaults"
	if err != nil {
		value = "127.0.0.1:8080"
	}
	return value
}

func explicitTestFixture() string {
	return "127.0.0.1:18080"
}
