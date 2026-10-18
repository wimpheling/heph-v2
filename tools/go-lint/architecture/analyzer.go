// Package architecture enforces Heph's application boundaries with type-aware checks.
package architecture

import (
	"go/ast"
	"go/types"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const module = "github.com/wimpheling/heph-v2"

var Analyzer = &analysis.Analyzer{
	Name: "hepharchitecture",
	Doc:  "enforce centralized environment reads and transport-independent core logic",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	checkArchitecture(pass)
	for _, file := range pass.Files {
		if ast.IsGenerated(file) {
			continue
		}
		name := filepath.ToSlash(pass.Fset.Position(file.Pos()).Filename)
		inConfig := filepath.Base(name) == "config.go" || strings.Contains("/"+name, "/config/")
		if !inConfig {
			ast.Inspect(file, func(node ast.Node) bool {
				ident, ok := node.(*ast.Ident)
				if !ok {
					return true
				}
				function, ok := pass.TypesInfo.Uses[ident].(*types.Func)
				if !ok || function.Pkg() == nil || function.Pkg().Path() != "os" {
					return true
				}
				switch function.Name() {
				case "Getenv", "LookupEnv", "Environ", "ExpandEnv":
					pass.Reportf(ident.Pos(), "read environment variables in config.go or config/ and pass typed configuration to other packages")
				}
				return true
			})
		}
		if pkg := pass.Pkg.Path(); pkg == module+"/internal/core" || strings.HasPrefix(pkg, module+"/internal/core/") {
			for _, spec := range file.Imports {
				path, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					return nil, err
				}
				if within(path, "net/http") || within(path, "connectrpc.com/connect") || within(path, module+"/gen") {
					pass.Reportf(spec.Pos(), "keep core independent of HTTP, ConnectRPC, and generated transport types; translate in internal/rpc")
				}
			}
		}
	}
	return nil, nil
}

func within(path, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}
