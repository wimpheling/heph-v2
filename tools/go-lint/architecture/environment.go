package architecture

import (
	"go/constant"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

const environmentDefaultMessage = "environment values must not have fallback defaults; require explicit configuration and return an error when missing"

// Track environment strings through SSA merges and local configuration helpers.
// This catches both assignment-based defaults and early-return fallback literals.
// It deliberately does not attempt whole-program semantic proof across packages.
func checkEnvironmentDefaults(pass *analysis.Pass) {
	functions := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs
	results := map[*ssa.Function]map[int]bool{}
	for changed := true; changed; {
		changed = false
		for _, function := range functions {
			for _, block := range function.Blocks {
				for _, instruction := range block.Instrs {
					ret, ok := instruction.(*ssa.Return)
					if !ok {
						continue
					}
					for index, value := range ret.Results {
						if environmentString(value, results, map[ssa.Value]bool{}) && !results[function][index] {
							if results[function] == nil {
								results[function] = map[int]bool{}
							}
							results[function][index] = true
							changed = true
						}
					}
				}
			}
		}
	}
	for _, function := range functions {
		for _, block := range function.Blocks {
			for _, instruction := range block.Instrs {
				switch value := instruction.(type) {
				case *ssa.Phi:
					if !environmentString(value, results, map[ssa.Value]bool{}) {
						continue
					}
					for _, edge := range value.Edges {
						if !environmentString(edge, results, map[ssa.Value]bool{}) {
							pass.Reportf(value.Pos(), environmentDefaultMessage)
							break
						}
					}
				case *ssa.Return:
					for index, result := range value.Results {
						literal, ok := result.(*ssa.Const)
						if results[function][index] && ok && literal.Value != nil && literal.Value.Kind() == constant.String && !returnsError(function, value) {
							pass.Reportf(value.Pos(), environmentDefaultMessage)
						}
					}
				}
			}
		}
	}
}

func returnsError(function *ssa.Function, ret *ssa.Return) bool {
	errorType := types.Universe.Lookup("error").Type()
	for index, result := range ret.Results {
		if !types.Identical(function.Signature.Results().At(index).Type(), errorType) {
			continue
		}
		literal, ok := result.(*ssa.Const)
		return !ok || !literal.IsNil()
	}
	return false
}

func environmentString(value ssa.Value, results map[*ssa.Function]map[int]bool, seen map[ssa.Value]bool) bool {
	if seen[value] || !types.Identical(value.Type().Underlying(), types.Typ[types.String]) {
		return false
	}
	seen[value] = true
	switch value := value.(type) {
	case *ssa.Call:
		return environmentCall(value, 0, results)
	case *ssa.Extract:
		call, ok := value.Tuple.(*ssa.Call)
		return ok && environmentCall(call, value.Index, results)
	case *ssa.ChangeType:
		return environmentString(value.X, results, seen)
	case *ssa.Convert:
		return environmentString(value.X, results, seen)
	case *ssa.Phi:
		for _, edge := range value.Edges {
			if environmentString(edge, results, seen) {
				return true
			}
		}
	}
	return false
}

func environmentCall(call *ssa.Call, index int, results map[*ssa.Function]map[int]bool) bool {
	function := call.Common().StaticCallee()
	if function == nil {
		return false
	}
	if object := function.Object(); object != nil && object.Pkg() != nil && object.Pkg().Path() == "os" {
		return index == 0 && (object.Name() == "Getenv" || object.Name() == "LookupEnv")
	}
	return results[function][index]
}
