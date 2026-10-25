// Package capacityguard decide si se puede retirar un conjunto de nodos de un pool.
// Se niega si algún subsistema queda por debajo de su capacidad mínima o si el lote
// supera el porcentaje máximo del pool: retirar capacidad despacio y nunca bajo el mínimo.
package capacityguard

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
)

// Node es un servidor del pool.
type Node struct {
	Name      string `json:"name"`
	Subsystem string `json:"subsystem"`
}

// Subsystem guarda la capacidad mínima que necesita para seguir atendiendo.
type Subsystem struct {
	Min int `json:"min"`
}

// Pool es la flota de una región.
type Pool struct {
	Region      string               `json:"region"`
	MaxBatchPct int                  `json:"maxBatchPct"` // máximo % del pool que se puede retirar de una vez
	Subsystems  map[string]Subsystem `json:"subsystems"`
	Nodes       []Node               `json:"nodes"`
}

// Impact es lo que pierde un subsistema con el plan.
type Impact struct {
	Subsystem string
	Before    int
	After     int
	Min       int
}

// OK dice si el subsistema sigue en o por encima de su mínimo.
func (i Impact) OK() bool { return i.After >= i.Min }

// Plan es el resultado de evaluar un retiro: el radio de impacto antes de ejecutar.
type Plan struct {
	Match    string
	Selected []Node
	Total    int
	MaxBatch int
	Impacts  []Impact // solo los subsistemas que pierden capacidad
}

// BatchOK dice si el lote cabe en el máximo por lote.
func (p Plan) BatchOK() bool { return len(p.Selected) <= p.MaxBatch }

// Approved es true si el lote cabe y ningún subsistema baja de su mínimo.
func (p Plan) Approved() bool {
	if len(p.Selected) == 0 || !p.BatchOK() {
		return false
	}
	for _, i := range p.Impacts {
		if !i.OK() {
			return false
		}
	}
	return true
}

// Load lee un pool desde JSON.
func Load(file string) (Pool, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return Pool{}, err
	}
	return Parse(b)
}

// Parse decodifica un pool.
func Parse(b []byte) (Pool, error) {
	var p Pool
	if err := json.Unmarshal(b, &p); err != nil {
		return Pool{}, fmt.Errorf("pool inválido: %w", err)
	}
	if p.MaxBatchPct <= 0 {
		p.MaxBatchPct = 10
	}
	return p, nil
}

// Evaluate calcula qué nodos selecciona el patrón (glob) y el radio de impacto. No retira nada.
func Evaluate(p Pool, match string) (Plan, error) {
	if _, err := path.Match(match, ""); err != nil {
		return Plan{}, fmt.Errorf("patrón inválido %q: %w", match, err)
	}
	plan := Plan{Match: match, Total: len(p.Nodes)}
	plan.MaxBatch = len(p.Nodes) * p.MaxBatchPct / 100
	before := map[string]int{}
	lost := map[string]int{}
	for _, n := range p.Nodes {
		before[n.Subsystem]++
		if ok, _ := path.Match(match, n.Name); ok {
			plan.Selected = append(plan.Selected, n)
			lost[n.Subsystem]++
		}
	}
	names := make([]string, 0, len(lost))
	for s := range lost {
		names = append(names, s)
	}
	sort.Strings(names)
	for _, s := range names {
		plan.Impacts = append(plan.Impacts, Impact{Subsystem: s, Before: before[s], After: before[s] - lost[s], Min: p.Subsystems[s].Min})
	}
	return plan, nil
}

// Apply devuelve el pool sin los nodos del plan. Solo se debe llamar si el plan está aprobado.
func Apply(p Pool, plan Plan) Pool {
	gone := map[string]bool{}
	for _, n := range plan.Selected {
		gone[n.Name] = true
	}
	out := p
	out.Nodes = nil
	for _, n := range p.Nodes {
		if !gone[n.Name] {
			out.Nodes = append(out.Nodes, n)
		}
	}
	return out
}
