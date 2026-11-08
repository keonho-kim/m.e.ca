// Purpose: Reject diagnostic text and code literals assigned after construction.
// Pattern: Typed field-assignment analysis.
// Usage: The diagnosticscheck analyzer checks mutable public diagnostic DTOs here.
// Related: devtools/internal/diagnosticcheck/check.go, devtools/internal/diagnosticcheck/testdata.
package diagnosticcheck

import (
	"go/ast"
	"go/constant"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

func checkAssignments(pass *analysis.Pass, file *ast.File) {
	ast.Inspect(file, func(node ast.Node) bool {
		assignment, ok := node.(*ast.AssignStmt)
		if !ok || len(assignment.Lhs) != len(assignment.Rhs) {
			return true
		}
		for i, left := range assignment.Lhs {
			field, ok := left.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			switch field.Sel.Name {
			case "Code", "ErrorCode", "Detail", "Next", "Error":
			default:
				continue
			}
			ast.Inspect(assignment.Rhs[i], func(value ast.Node) bool {
				literal, ok := value.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return true
				}
				if typed := pass.TypesInfo.Types[literal]; typed.Value != nil && constant.StringVal(typed.Value) != "" {
					pass.Reportf(literal.Pos(), "declare diagnostic text and codes in the central catalog")
				}
				return true
			})
		}
		return true
	})
}
