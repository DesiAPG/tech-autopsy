// pool-bench compara el costo de abrir conexiones contra reutilizarlas, con el mismo tráfico.
package main

import (
	"flag"
	"fmt"
	"os"

	poolbench "github.com/DesiAPG/tech-autopsy/diagnostico/004-cloudflare-pingora"
)

func main() {
	cfg := poolbench.Config{}
	flag.IntVar(&cfg.Requests, "n", 4000, "peticiones totales")
	flag.IntVar(&cfg.Concurrency, "c", 16, "peticiones en vuelo")
	flag.IntVar(&cfg.Workers, "workers", 16, "workers (pools) en el modo pool por worker")
	flag.Uint64Var(&cfg.Seed, "seed", 4, "semilla del reparto de peticiones")
	flag.Parse()

	fmt.Printf("%d peticiones · %d en vuelo · origen HTTPS local\n", cfg.Requests, cfg.Concurrency)
	fmt.Printf("%-16s %10s %9s %9s %9s\n", "estrategia", "handshakes", "reuso", "p50", "p95")
	for _, m := range []poolbench.Mode{poolbench.NoReuse, poolbench.PerWorker, poolbench.Shared} {
		r, err := poolbench.Run(m, cfg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("%-16s %10d %8.2f%% %9s %9s\n", m, r.Handshakes, r.Reuse(), r.P50.Round(10_000), r.P95.Round(10_000))
	}
}
