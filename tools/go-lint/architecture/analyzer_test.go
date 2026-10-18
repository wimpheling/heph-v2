package architecture

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestBoundaries(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer,
		"badenv", "allowed", "config", "rpc", "envdefaults/config",
		module+"/internal/core", module+"/internal/core/nested",
		module+"/internal/core/domain/valid", module+"/internal/core/domain/bad", module+"/internal/rpc",
	)
}
