// Package core contains transport-independent application logic.
package core

type SystemInfo struct {
	Name    string
	Version string
}

func GetSystemInfo() SystemInfo {
	return SystemInfo{Name: "heph-v2", Version: "0.1.0"}
}
