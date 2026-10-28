// Package contentgate reproduce el incidente de CrowdStrike del 19 de julio de 2024 (Channel File 291)
// con un mini intérprete de reglas: una plantilla define N campos, el código entrega M valores
// y una regla puede pedir cualquier campo. Incluye el validador que falló, el que lo habría frenado,
// un intérprete con chequeo de límites y un despliegue por anillos con rollback.
package contentgate

import (
	"errors"
	"fmt"
	"regexp"
)

// Wildcard es el criterio que acepta cualquier valor: el intérprete ni siquiera lee el campo.
const Wildcard = "*"

// Template es la definición de una plantilla (lo que dice el archivo de definiciones).
type Template struct {
	Name   string
	Fields int // campos que la definición declara
}

// Rule es una instancia: un criterio (regex o comodín) por campo.
type Rule struct {
	ID       string
	Criteria []string
}

// Event son los valores que el código del sensor entrega de verdad al intérprete.
type Event []string

// ErrOutOfBounds es lo que el intérprete seguro devuelve en lugar de leer fuera del arreglo.
var ErrOutOfBounds = errors.New("índice fuera de límites")

// ValidateAgainstDefinition es el validador del incidente: compara la regla contra la definición.
func ValidateAgainstDefinition(t Template, r Rule) error {
	if len(r.Criteria) > t.Fields {
		return fmt.Errorf("la regla usa %d campos; la plantilla define %d", len(r.Criteria), t.Fields)
	}
	return nil
}

// ValidateAgainstInputs es el validador corregido: ningún criterio que no sea comodín puede caer
// en un campo que el código no entrega.
func ValidateAgainstInputs(r Rule, provided int) error {
	for i, c := range r.Criteria {
		if c != Wildcard && i >= provided {
			return fmt.Errorf("usa el campo %d; el sensor entrega %d", i+1, provided)
		}
	}
	return nil
}

// Match es el intérprete con chequeo de límites: ante un campo que no existe devuelve un error
// controlado (el sensor seguiría vivo) en vez de leer memoria ajena.
func Match(r Rule, e Event) (bool, error) {
	for i, c := range r.Criteria {
		if c == Wildcard {
			continue
		}
		if i >= len(e) {
			return false, fmt.Errorf("campo %d (índice %d): %w", i+1, i, ErrOutOfBounds)
		}
		re, err := regexp.Compile(c)
		if err != nil {
			return false, err
		}
		if !re.MatchString(e[i]) {
			return false, nil
		}
	}
	return true, nil
}

// UnsafeMatch imita al intérprete del incidente: no revisa límites. En Go eso es un panic
// (index out of range); en un driver del kernel de Windows fue una pantalla azul.
func UnsafeMatch(r Rule, e Event) bool {
	for i, c := range r.Criteria {
		if c == Wildcard {
			continue
		}
		_ = e[i] // sin chequeo de límites
	}
	return true
}

// Ring es un anillo de despliegue.
type Ring struct {
	Name  string
	Hosts int
}

// Rollout es el resultado de desplegar una regla por anillos.
type Rollout struct {
	Reached    []string // anillos que la recibieron
	FailedRing string   // anillo donde falló ("" si llegó a todos)
	Crashed    int      // equipos que la recibieron con el intérprete inseguro y cayeron
	RolledBack bool
}

// Deploy promueve la regla anillo por anillo. En cada uno corre el intérprete inseguro contra un
// evento real (como en producción); si algún equipo cae, se detiene y hace rollback.
func Deploy(r Rule, e Event, rings []Ring) Rollout {
	var out Rollout
	for _, ring := range rings {
		out.Reached = append(out.Reached, ring.Name)
		if crashes(r, e) {
			out.FailedRing = ring.Name
			out.Crashed = ring.Hosts
			out.RolledBack = true
			return out
		}
	}
	return out
}

func crashes(r Rule, e Event) (crashed bool) {
	defer func() {
		if recover() != nil {
			crashed = true
		}
	}()
	UnsafeMatch(r, e)
	return false
}
