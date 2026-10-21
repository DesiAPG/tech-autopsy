// qs-mini compara replicar un log con secuencia contra replicar por timestamps, sobre una red que pierde mensajes.
package main

import (
	"flag"
	"fmt"
	"os"

	qslog "github.com/DesiAPG/tech-autopsy/anatomia/005-cloudflare-quicksilver"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "demo" {
		fmt.Fprintln(os.Stderr, "uso: qs-mini demo [-n cambios] [-replicas N] [-loss 0.05] [-batch 50] [-seed 5]")
		os.Exit(2)
	}
	fs := flag.NewFlagSet("demo", flag.ExitOnError)
	cfg := qslog.Config{}
	fs.IntVar(&cfg.Changes, "n", 10000, "cambios totales")
	fs.IntVar(&cfg.Replicas, "replicas", 3, "réplicas")
	fs.Float64Var(&cfg.Loss, "loss", 0.05, "probabilidad de perder cada mensaje")
	fs.IntVar(&cfg.Batch, "batch", 50, "cambios por lote (ventana de 500 ms)")
	fs.Int64Var(&cfg.Skew, "skew", 10, "desfase máximo entre relojes de escritores, en ms")
	fs.Uint64Var(&cfg.Seed, "seed", 5, "semilla")
	fs.Parse(os.Args[2:])

	log := qslog.Log(cfg)
	seq, ts := qslog.Sequence(cfg, log), qslog.Timestamps(cfg, log)
	ok := map[bool]string{true: "iguales ✓", false: "distintas ✖"}
	fmt.Printf("%d cambios · %d réplicas · %.0f %% de mensajes perdidos\n", cfg.Changes, cfg.Replicas, cfg.Loss*100)
	fmt.Printf("secuencia   huecos detectados %5d · reenviados %5d\n", seq.Gaps, seq.Resent)
	fmt.Printf("            réplicas %s\n", ok[seq.Identical])
	fmt.Printf("timestamps  entradas perdidas %5d · avisos 0\n", ts.Lost)
	fmt.Printf("            réplicas %s\n", ok[ts.Identical])
}
