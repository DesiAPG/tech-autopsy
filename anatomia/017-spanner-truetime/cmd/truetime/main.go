// truetime: el ejemplo del banco con marcas ingenuas y con TrueTime + commit wait, en tiempo simulado.
//
//	go run ./anatomia/017-spanner-truetime/cmd/truetime demo [-atraso 3ms] [-eps 4ms]
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tt "github.com/DesiAPG/tech-autopsy/anatomia/017-spanner-truetime"
)

func main() {
	fs := flag.NewFlagSet("demo", flag.ExitOnError)
	lag := fs.Duration("atraso", 3*time.Millisecond, "cuánto atrasa el reloj del servidor del pago")
	eps := fs.Duration("eps", 4*time.Millisecond, "incertidumbre que anuncia cada reloj (ε)")
	if len(os.Args) > 1 && os.Args[1] == "demo" {
		fs.Parse(os.Args[2:])
	} else {
		fs.Parse(os.Args[1:])
	}
	a := tt.Node{Name: "servidor del depósito", Skew: 0, Eps: *eps}
	b := tt.Node{Name: "servidor del pago", Skew: -*lag, Eps: *eps}
	start := 100 * time.Millisecond

	fmt.Printf("ahorros $50 · corriente $50 · el reloj del pago atrasa %v · ε = %v\n\n", *lag, *eps)

	dep := tt.CommitNaive(a, start)
	pay := tt.CommitNaive(b, start+time.Millisecond)
	show("marcas ingenuas (un solo reloj)", []tt.Op{{Name: "depósito +200", Account: "ahorros", Amount: 200, C: dep}, {Name: "pago −150", Account: "corriente", Amount: -150, C: pay}})

	dep = tt.CommitWait(a, start, 100*time.Microsecond)
	pay = tt.CommitWait(b, dep.Done+time.Millisecond, 100*time.Microsecond)
	show("TrueTime + commit wait", []tt.Op{{Name: "depósito +200", Account: "ahorros", Amount: 200, C: dep}, {Name: "pago −150", Account: "corriente", Amount: -150, C: pay}})
	fmt.Printf("espera a propósito: depósito %v · pago %v (al menos ~2ε)\n", dep.Wait, pay.Wait)
}

func show(title string, ops []tt.Op) {
	fmt.Println(title)
	for _, o := range ops {
		fmt.Printf("  %-14s marca %v\n", o.Name, o.C.TS)
	}
	worst := 1 << 30
	for _, o := range ops {
		if _, total := tt.Snapshot(ops, o.C.TS); total < worst {
			worst = total
		}
	}
	if tt.Penalty(ops) {
		fmt.Printf("  una lectura ve el pago sin el depósito: total $%d → MULTA ✗\n\n", worst)
	} else {
		fmt.Printf("  ninguna lectura queda en negativo (peor total $%d) ✓\n\n", worst)
	}
}
