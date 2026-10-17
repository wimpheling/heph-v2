package badenv

import . "os"

func dotted() {
	_ = Getenv("HEPH_LISTEN_ADDR") // want "read environment variables in config.go"
}
