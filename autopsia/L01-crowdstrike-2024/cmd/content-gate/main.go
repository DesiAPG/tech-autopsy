// content-gate reproduce el Channel File 291 de CrowdStrike (19-jul-2024): una plantilla que define
// 21 campos, un sensor que entrega 20 y la regla que pidió el campo 21.
//
//	content-gate demo
package main

import (
	"fmt"
	"os"

	cg "github.com/DesiAPG/tech-autopsy/autopsia/L01-crowdstrike-2024"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "demo" {
		fmt.Fprintln(os.Stderr, "uso: content-gate demo")
		os.Exit(2)
	}
	e := cg.SampleEvent()
	fmt.Printf("plantilla %s: %d campos definidos · %d entregados\n", cg.IPC.Name, cg.IPC.Fields, len(e))
	for _, r := range []cg.Rule{cg.RuleMarch, cg.RuleJuly} {
		fmt.Printf("\nregla %s\n", r.ID)
		check("validador viejo", cg.ValidateAgainstDefinition(cg.IPC, r))
		check("validador nuevo", cg.ValidateAgainstInputs(r, len(e)))
		_, err := cg.Match(r, e)
		check("intérprete", err)
		d := cg.Deploy(r, e, cg.Rings)
		if d.RolledBack {
			fmt.Printf("  %-16s ✕ cayó en %s (%d equipos) → rollback · %d de %d anillos\n", "anillos", d.FailedRing, d.Crashed, len(d.Reached), len(cg.Rings))
		} else {
			fmt.Printf("  %-16s ✓ llegó a %d de %d anillos\n", "anillos", len(d.Reached), len(cg.Rings))
		}
	}
}

func check(name string, err error) {
	if err != nil {
		fmt.Printf("  %-16s ✕ %v\n", name, err)
		return
	}
	fmt.Printf("  %-16s ✓\n", name)
}
