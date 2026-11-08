// Purpose: Reject product-owned diagnostic declarations outside their central module.
// Pattern: Typed Go syntax analysis.
// Usage: diagnosticscheck runs this analyzer before tests and release publication.
// Related: devtools/cmd/diagnosticscheck/main.go, internal/diagnostics/catalog.
package diagnosticcheck

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{Name: "diagnosticscheck", Doc: "require central product error and diagnostic declarations", Run: run}

func run(pass *analysis.Pass) (any, error) {
	central := strings.HasPrefix(pass.Pkg.Path(), "github.com/keonho-kim/meca/internal/diagnostics")
	for _, file := range pass.Files {
		if strings.HasSuffix(pass.Fset.Position(file.Pos()).Filename, "_test.go") || ast.IsGenerated(file) {
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selected, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			object := pass.TypesInfo.Uses[selected.Sel]
			if object == nil || object.Pkg() == nil {
				return true
			}
			owner, name := object.Pkg().Path(), object.Name()
			if !central {
				forbidden := owner == "fmt" && name == "Errorf" || owner == "errors" && name == "New"
				forbidden = forbidden || owner == "log" || owner == "log/slog" && (name == "New" || name == "NewJSONHandler" || name == "NewTextHandler" || strings.HasPrefix(name, "Log") || strings.HasPrefix(name, "Info") || strings.HasPrefix(name, "Warn") || strings.HasPrefix(name, "Error") || strings.HasPrefix(name, "Debug"))
				if forbidden {
					pass.Reportf(call.Pos(), "declare product errors and logs through internal/diagnostics")
				}
			}
			if strings.HasSuffix(owner, "/internal/diagnostics/catalog") && !central {
				if name == "Event" || name == "Message" {
					pass.Reportf(call.Pos(), "declare diagnostic templates and events in the central catalog")
				}
			}
			if !central && strings.HasSuffix(owner, "/internal/diagnostics/fault") {
				index := -1
				switch name {
				case "Errorf":
					index = 0
				case "New", "Invalid":
					index = 1
				}
				if index >= 0 && len(call.Args) > index {
					checkTemplate(pass, call.Args[index])
				}
				if name == "New" && len(call.Args) > 0 {
					if _, literal := call.Args[0].(*ast.BasicLit); literal {
						pass.Reportf(call.Args[0].Pos(), "declare error codes in the central catalog")
					}
				}
			}
			if !central && strings.HasSuffix(owner, "/internal/diagnostics") && name == "Log" && len(call.Args) > 1 {
				if _, literal := call.Args[1].(*ast.BasicLit); literal {
					pass.Reportf(call.Args[1].Pos(), "register diagnostic events in the central catalog")
				}
			}
			return true
		})
		if !central {
			checkAssignments(pass, file)
			ast.Inspect(file, func(node ast.Node) bool {
				composite, ok := node.(*ast.CompositeLit)
				if !ok {
					return true
				}
				named, ok := pass.TypesInfo.TypeOf(composite).(*types.Named)
				if !ok {
					return true
				}
				switch named.Obj().Name() {
				case "Error", "Failure", "InputError", "apiError", "SkillDiagnostic":
				default:
					return true
				}
				for _, element := range composite.Elts {
					field, ok := element.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					key, ok := field.Key.(*ast.Ident)
					if !ok {
						continue
					}
					switch key.Name {
					case "Code", "ErrorCode", "Detail", "Next", "Message":
						ast.Inspect(field.Value, func(value ast.Node) bool {
							if literal, ok := value.(*ast.BasicLit); ok {
								if valueType := pass.TypesInfo.TypeOf(literal); valueType != nil {
									if basic, ok := valueType.Underlying().(*types.Basic); ok && basic.Info()&types.IsString != 0 {
										pass.Reportf(literal.Pos(), "declare diagnostic text and codes in the central catalog")
									}
								}
							}
							return true
						})
					}
				}
				return true
			})
		}
	}
	return nil, nil
}

func checkTemplate(pass *analysis.Pass, expr ast.Expr) {
	selected, ok := expr.(*ast.SelectorExpr)
	if ok {
		if object := pass.TypesInfo.Uses[selected.Sel]; object != nil && object.Pkg() != nil && strings.HasSuffix(object.Pkg().Path(), "/internal/diagnostics/catalog") {
			return
		}
	}
	pass.Reportf(expr.Pos(), "declare error message templates in the central catalog")
}
