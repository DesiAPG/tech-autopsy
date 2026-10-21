// config-gate valida un archivo de features generado antes de propagarlo a la red.
// Sale con 1 si lo bloquea; en ese caso se mantiene la versión anterior.
package main

import (
	"flag"
	"fmt"
	"os"

	configgate "github.com/DesiAPG/tech-autopsy/autopsia/006-cloudflare-2025-feature-file"
)

func main() {
	if len(os.Args) < 3 || os.Args[1] != "check" {
		fmt.Fprintln(os.Stderr, "uso: config-gate check NUEVO.json -prev ULTIMO_BUENO.json [-max 200] [-growth 1.5]")
		os.Exit(2)
	}
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	prevPath := fs.String("prev", "", "última versión buena")
	lim := configgate.Limits{}
	fs.IntVar(&lim.MaxFeatures, "max", 200, "máximo de features que soporta el consumidor")
	fs.Float64Var(&lim.MaxGrowth, "growth", 1.5, "crecimiento máximo contra la versión anterior")
	fs.Parse(os.Args[3:])

	next, err := configgate.Load(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	var prev configgate.File
	if *prevPath != "" {
		if prev, err = configgate.Load(*prevPath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	fmt.Printf("v%d · %d features (antes v%d · %d)\n", next.Version, len(next.Features), prev.Version, len(prev.Features))
	cs := configgate.Validate(next, prev, lim)
	for _, c := range cs {
		fmt.Printf("%-12s %s %s\n", c.Rule, map[bool]string{true: "✓", false: "✖"}[c.OK], c.Detail)
	}
	if !configgate.Passed(cs) {
		fmt.Printf("BLOQUEADO · se mantiene v%d · no se propaga\n", prev.Version)
		os.Exit(1)
	}
	fmt.Printf("OK · v%d listo para desplegar por etapas\n", next.Version)
}
