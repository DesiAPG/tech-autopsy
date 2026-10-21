// Package configgate valida un archivo de configuración generado antes de propagarlo.
// Si algo no cuadra, no se publica y se mantiene la última versión buena ("fail small").
package configgate

import (
	"encoding/json"
	"fmt"
	"os"
)

// Feature es una columna del archivo de features.
type Feature struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// File es el archivo que se propaga a toda la red.
type File struct {
	Version  int       `json:"version"`
	Features []Feature `json:"features"`
}

// Limits son las reglas del gate.
type Limits struct {
	MaxFeatures int     // capacidad del consumidor (FL2 preasigna 200)
	MaxGrowth   float64 // crecimiento máximo permitido contra la versión anterior (1.5 = +50 %)
}

// Check es el resultado de una regla.
type Check struct {
	Rule   string
	OK     bool
	Detail string
}

// Load lee un archivo de features.
func Load(path string) (File, error) {
	var f File
	b, err := os.ReadFile(path)
	if err != nil {
		return f, err
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return f, fmt.Errorf("%s: %w", path, err)
	}
	return f, nil
}

// Validate aplica las reglas a next comparándolo con prev (la última versión buena).
func Validate(next, prev File, lim Limits) []Check {
	var out []Check

	schema := Check{Rule: "esquema", OK: len(next.Features) > 0}
	for _, f := range next.Features {
		if f.Name == "" || f.Type == "" {
			schema.OK = false
		}
	}
	schema.Detail = map[bool]string{true: "name y type en todas", false: "features vacías o incompletas"}[schema.OK]
	out = append(out, schema)

	out = append(out, Check{
		Rule:   "límite",
		OK:     len(next.Features) <= lim.MaxFeatures,
		Detail: fmt.Sprintf("%d / %d", len(next.Features), lim.MaxFeatures),
	})

	seen, dups := map[string]bool{}, 0
	for _, f := range next.Features {
		if seen[f.Name] {
			dups++
		}
		seen[f.Name] = true
	}
	out = append(out, Check{Rule: "duplicados", OK: dups == 0, Detail: fmt.Sprintf("%d", dups)})

	growth := 1.0
	if n := len(prev.Features); n > 0 {
		growth = float64(len(next.Features)) / float64(n)
	}
	out = append(out, Check{
		Rule:   "crecimiento",
		OK:     growth <= lim.MaxGrowth,
		Detail: fmt.Sprintf("×%.1f (máx ×%.1f)", growth, lim.MaxGrowth),
	})
	return out
}

// Passed dice si todas las reglas pasaron.
func Passed(cs []Check) bool {
	for _, c := range cs {
		if !c.OK {
			return false
		}
	}
	return true
}
