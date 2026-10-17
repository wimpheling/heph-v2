package main

import (
	"github.com/wimpheling/heph-v2/tools/go-lint/architecture"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(architecture.Analyzer)
}
