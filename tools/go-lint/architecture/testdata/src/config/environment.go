package config

import "os"

var address = os.Getenv("HEPH_LISTEN_ADDR")
