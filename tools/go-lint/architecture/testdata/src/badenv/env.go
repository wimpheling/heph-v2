package badenv

import environment "os"

func read() {
	_ = environment.Getenv("HEPH_LISTEN_ADDR")       // want "read environment variables in config.go"
	_, _ = environment.LookupEnv("HEPH_LISTEN_ADDR") // want "read environment variables in config.go"
	_ = environment.Environ()                        // want "read environment variables in config.go"
	_ = environment.ExpandEnv("$HEPH_LISTEN_ADDR")   // want "read environment variables in config.go"
}

var aliased = environment.Getenv // want "read environment variables in config.go"
