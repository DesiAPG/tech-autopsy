package regexguard

import (
	"regexp"
	"strings"
	"testing"
)

// Regex real de la regla del WAF que tumbó Cloudflare el 2019-07-02 (post-mortem oficial).
const cloudflare2019 = `(?:(?:\"|'|\]|\}|\\|\d|(?:nan|infinity|true|false|null|undefined|symbol|math)|\` + "`" + `|\-|\+)+[)]*;?((?:\s|-|~|!|{}|\|\||\+)*.*(?:.*=.*)))`

func rules(t *testing.T, pattern string) []Rule {
	t.Helper()
	fs, err := Lint(pattern)
	if err != nil {
		t.Fatalf("Lint(%q): %v", pattern, err)
	}
	var out []Rule
	for _, f := range fs {
		out = append(out, f.Rule)
	}
	return out
}

func has(rs []Rule, r Rule) bool {
	for _, x := range rs {
		if x == r {
			return true
		}
	}
	return false
}

func TestCloudflare2019IsFlagged(t *testing.T) {
	rs := rules(t, cloudflare2019)
	if !has(rs, RuleAdjacentWildcards) {
		t.Fatal("la regex de Cloudflare 2019 debería marcarse como comodines-seguidos")
	}
	if len(rs) != 1 {
		t.Fatalf("esperaba un solo hallazgo (sin duplicados), obtuve %v", rs)
	}
}

func TestDangerous(t *testing.T) {
	cases := map[string]Rule{
		`.*.*=.*`:       RuleAdjacentWildcards,
		`.*(?:.*=.*)`:   RuleAdjacentWildcards,
		`a.*\s*.+b`:     RuleAdjacentWildcards,
		`(a+)+$`:        RuleNestedQuantifier,
		`(?:a*b*)*c`:    RuleNestedQuantifier,
		`(a|a+)+`:       RuleNestedQuantifier,
		`^(\w+\s?)*$`:   RuleNestedQuantifier,
		`(?:[^"]*)+x`:   RuleNestedQuantifier,
		`x=.*;.*.*end$`: RuleAdjacentWildcards,
	}
	for p, want := range cases {
		if !has(rules(t, p), want) {
			t.Errorf("%q: esperaba %s, obtuve %v", p, want, rules(t, p))
		}
	}
}

func TestSafe(t *testing.T) {
	for _, p := range []string{
		`^[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}$`,
		`.*=.*`,
		`foo=[^;]*;`,
		`(ab*)+`,
		`\d{4}-\d{2}-\d{2}`,
		`<script[^>]*>`,
	} {
		if rs := rules(t, p); len(rs) != 0 {
			t.Errorf("%q: no debería marcarse, obtuve %v", p, rs)
		}
	}
}

func TestUnsupportedSyntax(t *testing.T) {
	if _, err := Lint(`(?<=a)b`); err == nil {
		t.Fatal("un lookbehind debería devolver error (Go no lo soporta)")
	}
}

// El motor con backtracking crece de forma superlineal; regexp de Go no.
func TestBacktrackGrowsSuperlinearly(t *testing.T) {
	_, s20 := Backtrack(`.*.*=.*`, strings.Repeat("x", 20))
	_, s40 := Backtrack(`.*.*=.*`, strings.Repeat("x", 40))
	if s40 < 3*s20 {
		t.Fatalf("esperaba crecimiento superlineal: 20→%d pasos, 40→%d pasos", s20, s40)
	}
	for _, in := range []string{"x=x", "x=xxxx", strings.Repeat("x", 20)} {
		got, _ := Backtrack(`.*.*=.*`, in)
		want := regexp.MustCompile(`.*.*=.*`).MatchString(in)
		if got != want {
			t.Errorf("Backtrack(%q) = %v, regexp = %v", in, got, want)
		}
	}
}
