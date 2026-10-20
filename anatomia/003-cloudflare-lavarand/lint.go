package entropymix

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
)

// sensitive son los nombres que delatan un secreto: si una función o una
// variable se llama así y usa math/rand, el valor es predecible.
var sensitive = regexp.MustCompile(`(?i)token|secret|key|password|passwd|nonce|salt|session|otp|csrf|apikey`)

// Finding es un uso de math/rand para algo que parece un secreto.
type Finding struct {
	Pos  token.Position
	Call string // p. ej. "rand.Intn"
	Why  string // el nombre que lo delató
}

func (f Finding) String() string {
	return fmt.Sprintf("%s: ✖ math-rand-para-secretos · %s en %q", f.Pos, f.Call, f.Why)
}

// Lint revisa un archivo Go y reporta cada llamada a math/rand (o math/rand/v2)
// dentro de una función cuyo nombre, o el de alguna variable que asigna, parece
// un secreto. Es una heurística: no sigue los valores entre funciones.
func Lint(path string, src any) ([]Finding, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, 0)
	if err != nil {
		return nil, err
	}
	alias := map[string]bool{}
	for _, imp := range f.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		if p != "math/rand" && p != "math/rand/v2" {
			continue
		}
		name := "rand"
		if imp.Name != nil {
			name = imp.Name.Name
		}
		alias[name] = true
	}
	if len(alias) == 0 {
		return nil, nil
	}

	var out []Finding
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		why := ""
		if sensitive.MatchString(fn.Name.Name) {
			why = fn.Name.Name
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if why != "" {
				return false
			}
			if as, ok := n.(*ast.AssignStmt); ok {
				for _, l := range as.Lhs {
					if id, ok := l.(*ast.Ident); ok && sensitive.MatchString(id.Name) {
						why = id.Name
					}
				}
			}
			if vs, ok := n.(*ast.ValueSpec); ok {
				for _, id := range vs.Names {
					if sensitive.MatchString(id.Name) {
						why = id.Name
					}
				}
			}
			return true
		})
		if why == "" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if x, ok := sel.X.(*ast.Ident); ok && alias[x.Name] {
				out = append(out, Finding{fset.Position(call.Pos()), x.Name + "." + sel.Sel.Name, why})
			}
			return true
		})
	}
	return out, nil
}
