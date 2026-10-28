// capacity-guard retira nodos de un pool solo si ningún subsistema queda por debajo de su mínimo
// y el lote no supera el máximo por lote. Con --dry-run solo muestra el radio de impacto.
// Sin --pool usa el pool de ejemplo de us-east-1 (el del caso). Sale con 1 si rechaza el plan.
package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	capacityguard "github.com/DesiAPG/tech-autopsy/autopsia/007-aws-2017-s3"
)

//go:embed demo-pool.json
var demoPool []byte

func main() {
	match := flag.String("match", "", "patrón (glob) de los nodos a retirar, p. ej. 's3-billing-0[1-3]'")
	poolPath := flag.String("pool", "", "pool en JSON (por defecto, el ejemplo de us-east-1)")
	dry := flag.Bool("dry-run", false, "solo mostrar el radio de impacto, no retirar nada")
	flag.Parse()
	if *match == "" {
		fmt.Fprintln(os.Stderr, "uso: capacity-guard --match PATRÓN [--pool pool.json] [--dry-run]")
		os.Exit(2)
	}

	pool, err := capacityguard.Parse(demoPool)
	if *poolPath != "" {
		pool, err = capacityguard.Load(*poolPath)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	plan, err := capacityguard.Evaluate(pool, *match)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	fmt.Printf("plan: retirar %d de %d nodos (lote máx. %d)\n", len(plan.Selected), plan.Total, plan.MaxBatch)
	for _, i := range plan.Impacts {
		mark := "✓"
		if !i.OK() {
			mark = "✕"
		}
		fmt.Printf("%s %s: %d → %d (mínimo %d)\n", mark, i.Subsystem, i.Before, i.After, i.Min)
	}
	switch {
	case len(plan.Selected) == 0:
		fmt.Println("RECHAZADO · el patrón no selecciona ningún nodo")
		os.Exit(1)
	case !plan.Approved():
		fmt.Println("RECHAZADO · no se retiró nada")
		os.Exit(1)
	case *dry:
		fmt.Println("OK · dry-run: no se retiró nada")
	default:
		next := capacityguard.Apply(pool, plan)
		if *poolPath != "" {
			b, _ := json.MarshalIndent(next, "", "  ")
			if err := os.WriteFile(*poolPath, append(b, '\n'), 0o644); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
		}
		fmt.Printf("OK · retirados %d nodos · quedan %d\n", len(plan.Selected), len(next.Nodes))
	}
}
