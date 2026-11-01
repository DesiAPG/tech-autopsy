// ring: la demo del caso 013 (Slack, 2023). Reparte canales con un anillo de hashing consistente,
// tumba un servidor y compara cuántos canales cambian de dueño frente a hash % N.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	ring "github.com/DesiAPG/tech-autopsy/anatomia/013-slack-hashing-consistente"
)

func pct(n, total int) string {
	return strings.Replace(fmt.Sprintf("%.1f %%", float64(n)*100/float64(total)), ".", ",", 1)
}

func main() {
	fs := flag.NewFlagSet("ring", flag.ExitOnError)
	canales := fs.Int("canales", 100000, "canales a repartir")
	nserv := fs.Int("servidores", 6, "Channel Servers en el anillo")
	vnodes := fs.Int("vnodes", 160, "puntos por servidor en el anillo")
	cae := fs.String("cae", "cs-4", "servidor que se cae")
	if len(os.Args) < 2 || os.Args[1] != "demo" {
		fmt.Println("uso: ring demo [-canales N] [-servidores N] [-vnodes N] [-cae cs-4]")
		os.Exit(2)
	}
	fs.Parse(os.Args[2:])

	var servers, rest []string
	for i := 1; i <= *nserv; i++ {
		s := fmt.Sprintf("cs-%d", i)
		servers = append(servers, s)
		if s != *cae {
			rest = append(rest, s)
		}
	}
	r := ring.New(*vnodes, servers...)
	before := make([]string, *canales)
	for i := range before {
		before[i] = r.Owner(fmt.Sprintf("C%07d", i))
	}
	r.Remove(*cae)

	movedRing, movedMod := 0, 0
	for i, was := range before {
		k := fmt.Sprintf("C%07d", i)
		if r.Owner(k) != was {
			movedRing++
		}
		if ring.ModN(k, servers) != ring.ModN(k, rest) {
			movedMod++
		}
	}
	fmt.Println("$ ring demo")
	fmt.Printf("%d canales · %d servidores\n", *canales, *nserv)
	fmt.Printf("cae %s:\n", *cae)
	fmt.Printf("  anillo    mueve %s ✓\n", pct(movedRing, *canales))
	fmt.Printf("  hash %% N  mueve %s ✗\n", pct(movedMod, *canales))
}
