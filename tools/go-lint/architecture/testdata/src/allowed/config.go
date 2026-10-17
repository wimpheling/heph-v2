package allowed

import "os"

func read() {
	_ = os.Getenv("HEPH_LISTEN_ADDR")
	_, _ = os.LookupEnv("HEPH_LISTEN_ADDR")
	_ = os.Environ()
	_ = os.ExpandEnv("$HEPH_LISTEN_ADDR")
}
