package architecture

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const domainPath = module + "/internal/core/domain"
const operationPath = module + "/internal/core/operation"
const authorizationPath = module + "/internal/core/authorization"
const platformPath = module + "/internal/rpc/platform"

func function(pass *analysis.Pass, expr ast.Expr) *types.Func {
	switch e := expr.(type) {
	case *ast.Ident:
		f, _ := pass.TypesInfo.Uses[e].(*types.Func)
		return f
	case *ast.SelectorExpr:
		f, _ := pass.TypesInfo.Uses[e.Sel].(*types.Func)
		return f
	case *ast.IndexExpr:
		return function(pass, e.X)
	case *ast.IndexListExpr:
		return function(pass, e.X)
	case *ast.ParenExpr:
		return function(pass, e.X)
	}
	return nil
}
func isFunction(f *types.Func, path, name string) bool {
	return f != nil && f.Pkg() != nil && f.Pkg().Path() == path && f.Name() == name
}
func operationFactory(f *types.Func) bool {
	if f == nil {
		return false
	}
	sig, ok := f.Type().(*types.Signature)
	if !ok || sig.Results().Len() != 1 {
		return false
	}
	pointer, ok := types.Unalias(sig.Results().At(0).Type()).(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := types.Unalias(pointer.Elem()).(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == operationPath && named.Obj().Name() == "Operation"
}
func directArgument(parents []ast.Node, index int, pass *analysis.Pass, path, name string) bool {
	if len(parents) < 2 {
		return false
	}
	call, ok := parents[len(parents)-2].(*ast.CallExpr)
	return ok && index < len(call.Args) && call.Args[index] == parents[len(parents)-1] && isFunction(function(pass, call.Fun), path, name)
}
func requestType(typ types.Type) bool {
	pointer, ok := types.Unalias(typ).(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := types.Unalias(pointer.Elem()).(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == "connectrpc.com/connect" && named.Obj().Name() == "Request"
}
func allowedDomainCall(f *types.Func) bool {
	if f == nil || f.Pkg() == nil {
		return false
	}
	path, name := f.Pkg().Path(), f.Name()
	if path == "errors" {
		return name == "Is" || name == "As"
	}
	if path == module+"/internal/core" {
		return name == "GetSystemInfo"
	}
	if path == operationPath {
		return name == "Define" || name == "Require" || name == "Public"
	}
	if path != authorizationPath {
		return false
	}
	switch name {
	case "Create", "Get", "List", "Revoke", "ActiveGrants":
		sig := f.Type().(*types.Signature)
		if sig.Recv() == nil {
			return false
		}
		typ := sig.Recv().Type()
		if p, ok := typ.(*types.Pointer); ok {
			typ = p.Elem()
		}
		named, ok := typ.(*types.Named)
		return ok && (named.Obj().Name() == "Service" || (named.Obj().Name() == "GrantReader" && name == "ActiveGrants"))
	case "CreateScope", "StoredScope", "ListScope", "CanDelegate":
		return true
	}
	return false
}
func checkArchitecture(pass *analysis.Pass) {
	rpcMethods := map[string]token.Pos{}
	bindings := map[string][]token.Pos{}
	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename
		if ast.IsGenerated(file) || strings.HasSuffix(filename, "_test.go") {
			continue
		}
		inDomain := within(pass.Pkg.Path(), domainPath)
		for _, decl := range file.Decls {
			if global, ok := decl.(*ast.GenDecl); ok && inDomain && global.Tok == token.VAR {
				pass.Reportf(global.Pos(), "domain packages cannot declare mutable global state")
			}
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			obj, _ := pass.TypesInfo.Defs[fn.Name].(*types.Func)
			if inDomain {
				if fn.Name.Name == "init" {
					pass.Reportf(fn.Pos(), "domain packages cannot run unguarded initialization")
				}
				if operationFactory(obj) {
					valid := false
					if len(fn.Body.List) == 1 {
						if ret, ok := fn.Body.List[0].(*ast.ReturnStmt); ok && len(ret.Results) == 1 {
							if call, ok := ret.Results[0].(*ast.CallExpr); ok && isFunction(function(pass, call.Fun), operationPath, "Define") && len(call.Args) == 3 {
								if policy, ok := call.Args[1].(*ast.CallExpr); ok {
									p := function(pass, policy.Fun)
									valid = isFunction(p, operationPath, "Public") || isFunction(p, operationPath, "Require")
								}
							}
						}
					}
					if !valid {
						pass.Reportf(fn.Pos(), "domain operation factories must only return operation.Define with an explicit policy")
					}
				} else if obj != nil && obj.Exported() {
					pass.Reportf(fn.Pos(), "domain business handlers must be private")
				}
			}
			if pass.Pkg.Path() == module+"/internal/rpc" && fn.Recv != nil && obj != nil {
				sig := obj.Type().(*types.Signature)
				if sig.Params().Len() == 2 && requestType(sig.Params().At(1).Type()) {
					rpcMethods[fn.Name.Name] = fn.Pos()
					valid := false
					if len(fn.Body.List) == 1 {
						if ret, ok := fn.Body.List[0].(*ast.ReturnStmt); ok && len(ret.Results) == 1 {
							if call, ok := ret.Results[0].(*ast.CallExpr); ok {
								f := function(pass, call.Fun)
								valid = isFunction(f, platformPath, "Call")
							}
						}
					}
					if !valid {
						pass.Reportf(fn.Pos(), "RPC adapters must delegate directly to their typed endpoint binding")
					}
				}
			}
		}
		parents := []ast.Node{}
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				parents = parents[:len(parents)-1]
				return true
			}
			parents = append(parents, node)
			switch n := node.(type) {
			case *ast.FuncLit:
				if inDomain {
					pass.Reportf(n.Pos(), "anonymous functions are prohibited in domain handlers")
				}
			case *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.GoStmt, *ast.SendStmt, *ast.SelectStmt, *ast.DeferStmt:
				if inDomain {
					pass.Reportf(node.Pos(), "loops, switches, concurrency, and deferred calls are prohibited in domain handlers")
				}
			case *ast.UnaryExpr:
				if inDomain && n.Op != token.NOT && n.Op != token.AND {
					pass.Reportf(n.Pos(), "domain computations belong in approved services")
				}
			case *ast.IncDecStmt, *ast.SliceExpr, *ast.TypeAssertExpr:
				if inDomain {
					pass.Reportf(node.Pos(), "domain computations belong in approved services")
				}
			case *ast.AssignStmt:
				if inDomain && n.Tok != token.ASSIGN && n.Tok != token.DEFINE {
					pass.Reportf(node.Pos(), "domain computations belong in approved services")
				}
			case *ast.IndexExpr:
				if inDomain && function(pass, n) == nil && !pass.TypesInfo.Types[n].IsType() {
					pass.Reportf(node.Pos(), "domain computations belong in approved services")
				}
			case *ast.BinaryExpr:
				if inDomain {
					allowed := n.Op == token.LAND || n.Op == token.LOR
					if n.Op == token.EQL || n.Op == token.NEQ {
						var value ast.Expr
						if id, ok := n.X.(*ast.Ident); ok && id.Name == "nil" {
							value = n.Y
						}
						if id, ok := n.Y.(*ast.Ident); ok && id.Name == "nil" {
							value = n.X
						}
						if value != nil {
							errType := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
							typ := pass.TypesInfo.TypeOf(value)
							allowed = typ != nil && types.Implements(typ, errType)
						}
					}
					if !allowed {
						pass.Reportf(n.Pos(), "domain computations belong in approved services; only boolean operators and error comparisons are allowed")
					}
				}
			case *ast.CallExpr:
				f := function(pass, n.Fun)
				if inDomain && !allowedDomainCall(f) {
					pass.Reportf(n.Pos(), "domain calls require an explicitly approved service, port, or helper")
				}
				if isFunction(f, operationPath, "Define") && !inDomain && pass.Pkg.Path() != operationPath {
					pass.Reportf(n.Pos(), "define business operations only in internal/core/domain")
				}
				if inDomain && isFunction(f, operationPath, "Define") && len(n.Args) == 3 {
					handler := function(pass, n.Args[2])
					if handler == nil || handler.Pkg() == nil || !within(handler.Pkg().Path(), domainPath) || operationFactory(handler) {
						pass.Reportf(n.Args[2].Pos(), "operation definitions require an actual private domain handler reference")
					}
				}
				if inDomain && isFunction(f, operationPath, "Require") && len(n.Args) > 0 {
					var obj types.Object
					switch action := n.Args[0].(type) {
					case *ast.SelectorExpr:
						obj = pass.TypesInfo.Uses[action.Sel]
					case *ast.Ident:
						obj = pass.TypesInfo.Uses[action]
					}
					constant, ok := obj.(*types.Const)
					if !ok || constant.Pkg() == nil || constant.Pkg().Path() != authorizationPath {
						pass.Reportf(n.Args[0].Pos(), "policies require a registered authorization action constant")
					}
					for _, callback := range n.Args[1:] {
						if !allowedDomainCall(function(pass, callback)) {
							pass.Reportf(callback.Pos(), "policies require approved resolver and additional check references")
						}
					}
				}
				if isFunction(f, platformPath, "Bind") {
					if len(n.Args) == 5 {
						method, ok := n.Args[1].(*ast.CallExpr)
						if !ok || len(method.Args) != 1 {
							pass.Reportf(n.Pos(), "endpoint bindings require a concrete method descriptor")
						} else if lit, ok := method.Args[0].(*ast.BasicLit); ok {
							name, _ := strconv.Unquote(lit.Value)
							bindings[name] = append(bindings[name], n.Pos())
						} else {
							pass.Reportf(n.Pos(), "endpoint bindings require a concrete method name")
						}
						op, ok := n.Args[2].(*ast.CallExpr)
						if !ok || function(pass, op.Fun) == nil || !within(function(pass, op.Fun).Pkg().Path(), domainPath) {
							pass.Reportf(n.Pos(), "bind RPCs to a domain operation factory")
						}
						for _, convert := range n.Args[3:] {
							if function(pass, convert) == nil {
								pass.Reportf(convert.Pos(), "endpoint converters require actual Go function references")
							}
						}
					}
				}
			case *ast.CompositeLit:
				typ := pass.TypesInfo.TypeOf(n)
				if named, ok := types.Unalias(typ).(*types.Named); ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == operationPath && pass.Pkg.Path() != operationPath {
					pass.Reportf(n.Pos(), "construct operations and policies only through explicit domain operation definitions")
				}
			case *ast.Ident:
				f, _ := pass.TypesInfo.Uses[n].(*types.Func)
				if f == nil || f.Pkg() == nil {
					break
				}
				// Walk through selectors to find the expression that owns this function reference.
				refs := parents
				for len(refs) > 1 {
					current := refs[len(refs)-1]
					unwrap := false
					switch parent := refs[len(refs)-2].(type) {
					case *ast.SelectorExpr:
						unwrap = parent.Sel == current
					case *ast.ParenExpr:
						unwrap = parent.X == current
					case *ast.IndexExpr:
						unwrap = parent.X == current
					case *ast.IndexListExpr:
						unwrap = parent.X == current
					}
					if !unwrap {
						break
					}
					refs = refs[:len(refs)-1]
				}
				if inDomain && !within(f.Pkg().Path(), domainPath) {
					called := false
					if len(refs) > 1 {
						if call, ok := refs[len(refs)-2].(*ast.CallExpr); ok {
							called = call.Fun == refs[len(refs)-1]
						}
					}
					// A generic function name sits inside its instantiation before the call.
					if len(refs) > 2 {
						if _, ok := refs[len(refs)-2].(*ast.IndexExpr); ok {
							if call, ok := refs[len(refs)-3].(*ast.CallExpr); ok {
								called = call.Fun == refs[len(refs)-2]
							}
						}
					}
					policyRef := false
					for _, index := range []int{1, 2, 3} {
						policyRef = policyRef || directArgument(refs, index, pass, operationPath, "Require")
					}
					if !called && !policyRef {
						pass.Reportf(n.Pos(), "domain function references are only allowed in calls and approved operation definitions")
					}
				}
				if within(f.Pkg().Path(), domainPath) && !operationFactory(f) {
					if !directArgument(refs, 2, pass, operationPath, "Define") {
						pass.Reportf(n.Pos(), "raw domain handlers may only be referenced by operation.Define")
					}
				}
				if strings.HasPrefix(f.Pkg().Path(), module+"/gen/") && strings.HasPrefix(f.Name(), "New") && strings.HasSuffix(f.Name(), "ServiceHandler") {
					if !directArgument(refs, 3, pass, platformPath, "Register") {
						pass.Reportf(n.Pos(), "register RPC handlers through the platform")
					}
				}
				if f.Pkg().Path() == authorizationPath && (f.Name() == "Create" || f.Name() == "Revoke") && !inDomain && pass.Pkg.Path() != authorizationPath {
					pass.Reportf(n.Pos(), "grant mutations must use guarded domain operations")
				}
			}
			return true
		})
	}
	if pass.Pkg.Path() == module+"/internal/rpc" {
		for name, pos := range rpcMethods {
			if len(bindings[name]) != 1 {
				pass.Reportf(pos, "RPC %s requires exactly one typed endpoint binding", name)
			}
		}
		for name, positions := range bindings {
			if len(positions) > 1 {
				for _, pos := range positions[1:] {
					pass.Reportf(pos, "duplicate typed endpoint binding for %s", name)
				}
			}
		}
	}
}
