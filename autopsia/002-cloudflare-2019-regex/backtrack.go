package regexguard

// Backtrack es un motor de regex mínimo con backtracking (literales, `.`, `*`,
// `^` y `$`), al estilo del clásico de Kernighan y Pike. Se comporta como PCRE
// para estas construcciones: cuando una opción falla, retrocede y prueba otra.
// Sirve para medir cómo crece el trabajo; no es un motor para producción.
//
// Devuelve si hay coincidencia y cuántos pasos dio el motor.
func Backtrack(pattern, text string) (bool, int) {
	m := &bt{re: []byte(pattern), text: []byte(text)}
	if len(m.re) > 0 && m.re[0] == '^' {
		return m.here(1, 0), m.steps
	}
	for i := 0; i <= len(m.text); i++ {
		if m.here(0, i) {
			return true, m.steps
		}
	}
	return false, m.steps
}

type bt struct {
	re, text []byte
	steps    int
}

func (m *bt) here(r, t int) bool {
	m.steps++
	if r == len(m.re) {
		return true
	}
	if r+1 < len(m.re) && m.re[r+1] == '*' {
		return m.star(m.re[r], r+2, t)
	}
	if m.re[r] == '$' && r+1 == len(m.re) {
		return t == len(m.text)
	}
	if t < len(m.text) && (m.re[r] == '.' || m.re[r] == m.text[t]) {
		return m.here(r+1, t+1)
	}
	return false
}

// star es codicioso: consume todo lo que puede y luego retrocede de a un carácter.
func (m *bt) star(c byte, r, t int) bool {
	end := t
	for end < len(m.text) && (c == '.' || m.text[end] == c) {
		end++
	}
	for i := end; i >= t; i-- {
		if m.here(r, i) {
			return true
		}
	}
	return false
}
