package allowed

func Getenv(key string) string { return key }

func unrelated() {
	_ = Getenv("not an environment read")
}
