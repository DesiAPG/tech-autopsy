// Package regexguard detecta expresiones regulares con riesgo de backtracking
// catastrófico, como la que tumbó Cloudflare el 2 de julio de 2019.
package regexguard

import (
	"fmt"
	"regexp/syntax"
	"strings"
)

// Rule identifica el tipo de patrón peligroso encontrado.
type Rule string

const (
	// RuleAdjacentWildcards: dos comodines sin límite seguidos (`.*.*`), separados
	// a lo sumo por cosas que pueden coincidir con la cadena vacía.
	RuleAdjacentWildcards Rule = "comodines-seguidos"
	// RuleNestedQuantifier: un cuantificador sin límite dentro de otro, como `(a+)+`.
	RuleNestedQuantifier Rule = "cuantificador-anidado"
)

// Finding es un problema detectado en un patrón.
type Finding struct {
	Rule    Rule
	Message string
	// Fragment es el trozo del patrón donde está el problema.
	Fragment string
}

// Lint analiza un patrón con sintaxis Perl/PCRE y devuelve los hallazgos.
// Devuelve error si el patrón no se puede interpretar (por ejemplo, usa
// lookarounds o backreferences, que Go no soporta).
func Lint(pattern string) ([]Finding, error) {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return nil, fmt.Errorf("no se pudo analizar el patrón: %w", err)
	}
	var out []Finding
	walk(re, func(n *syntax.Regexp) {
		if unbounded(n) && riskyIteration(n.Sub[0]) {
			out = append(out, Finding{
				Rule:     RuleNestedQuantifier,
				Message:  "cuantificador sin límite dentro de otro: el número de formas de partir el texto crece de forma exponencial",
				Fragment: pretty(n),
			})
		}
		if n.Op == syntax.OpConcat {
			out = append(out, adjacentWildcards(flatten(n.Sub))...)
		}
	})
	return dedupe(out), nil
}

// dedupe quita hallazgos repetidos: un mismo par de comodines aparece en la
// concatenación interna y en la externa.
func dedupe(fs []Finding) []Finding {
	seen := map[Finding]bool{}
	var out []Finding
	for _, f := range fs {
		if !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	return out
}

// pretty muestra un nodo como lo escribiría una persona (`.*` en vez de `(?-s:.*)`).
func pretty(n *syntax.Regexp) string {
	return strings.NewReplacer("(?-s:.*)", ".*", "(?-s:.+)", ".+", "(?-s:.)", ".").Replace(n.String())
}

// adjacentWildcards busca dos repeticiones sin límite de un comodín amplio
// (`.` o una clase de caracteres grande) que puedan competir por los mismos caracteres.
func adjacentWildcards(items []*syntax.Regexp) []Finding {
	var out []Finding
	prev := -1
	for i, it := range items {
		switch {
		case wideUnbounded(it):
			if prev >= 0 {
				out = append(out, Finding{
					Rule:     RuleAdjacentWildcards,
					Message:  "dos comodines sin límite seguidos: el motor prueba cada forma de repartir el texto entre ellos (el patrón de Cloudflare 2019: .*(?:.*=.*))",
					Fragment: pretty(items[prev]) + " … " + pretty(it),
				})
			}
			prev = i
		case nullable(it):
			// No consume nada obligatorio: los dos comodines siguen compitiendo.
		default:
			prev = -1
		}
	}
	return out
}

// flatten aplana concatenaciones y grupos para ver la secuencia real de piezas.
func flatten(subs []*syntax.Regexp) []*syntax.Regexp {
	var out []*syntax.Regexp
	for _, s := range subs {
		switch {
		case s.Op == syntax.OpConcat:
			out = append(out, flatten(s.Sub)...)
		case s.Op == syntax.OpCapture && s.Sub[0].Op == syntax.OpConcat:
			out = append(out, flatten(s.Sub[0].Sub)...)
		case s.Op == syntax.OpCapture:
			out = append(out, flatten(s.Sub)...)
		default:
			out = append(out, s)
		}
	}
	return out
}

func walk(n *syntax.Regexp, f func(*syntax.Regexp)) {
	f(n)
	for _, s := range n.Sub {
		walk(s, f)
	}
}

func unbounded(n *syntax.Regexp) bool {
	switch n.Op {
	case syntax.OpStar, syntax.OpPlus:
		return true
	case syntax.OpRepeat:
		return n.Max == -1
	}
	return false
}

// riskyIteration indica si el cuerpo de una repetición sin límite puede, a su
// vez, repetirse sin límite: entonces hay muchísimas formas de repartir el texto
// entre iteraciones (`(a+)+`, `(a*b*)*`, `(a|a+)+`).
func riskyIteration(n *syntax.Regexp) bool {
	for n.Op == syntax.OpCapture {
		n = n.Sub[0]
	}
	switch n.Op {
	case syntax.OpStar, syntax.OpPlus, syntax.OpRepeat:
		return unbounded(n)
	case syntax.OpConcat:
		hasRepeat := false
		for _, s := range n.Sub {
			switch {
			case unbounded(s):
				hasRepeat = true
			case nullable(s):
			default:
				return false
			}
		}
		return hasRepeat
	case syntax.OpAlternate:
		for _, s := range n.Sub {
			if riskyIteration(s) {
				return true
			}
		}
	}
	return false
}

// wideUnbounded: `.*`, `.+`, `[^x]*` y similares, que aceptan casi cualquier carácter.
func wideUnbounded(n *syntax.Regexp) bool {
	if !unbounded(n) {
		return false
	}
	s := n.Sub[0]
	if s.Op == syntax.OpCapture {
		s = s.Sub[0]
	}
	switch s.Op {
	case syntax.OpAnyChar, syntax.OpAnyCharNotNL:
		return true
	case syntax.OpCharClass:
		return classSize(s) > 1000
	}
	return false
}

func classSize(n *syntax.Regexp) int {
	total := 0
	for i := 0; i+1 < len(n.Rune); i += 2 {
		total += int(n.Rune[i+1]-n.Rune[i]) + 1
	}
	return total
}

func nullable(n *syntax.Regexp) bool {
	switch n.Op {
	case syntax.OpEmptyMatch, syntax.OpStar, syntax.OpQuest,
		syntax.OpBeginLine, syntax.OpEndLine, syntax.OpBeginText, syntax.OpEndText,
		syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		return true
	case syntax.OpRepeat:
		return n.Min == 0
	case syntax.OpCapture:
		return nullable(n.Sub[0])
	case syntax.OpConcat:
		for _, s := range n.Sub {
			if !nullable(s) {
				return false
			}
		}
		return true
	case syntax.OpAlternate:
		for _, s := range n.Sub {
			if nullable(s) {
				return true
			}
		}
	}
	return false
}
