// coalesce-bench: muchas peticiones iguales a la vez contra una "base" lenta,
// sin agrupar y con request coalescing (singleflight), como los data services de Discord.
package main

import (
	"flag"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// grupo es un singleflight mínimo (la misma idea que golang.org/x/sync/singleflight, sin dependencias):
// la primera llamada con una clave hace el trabajo; las que llegan mientras está en vuelo esperan y comparten el resultado.
type grupo struct {
	mu      sync.Mutex
	enVuelo map[string]*llamada
}

type llamada struct {
	wg  sync.WaitGroup
	val string
}

func (g *grupo) Do(clave string, fn func() string) string {
	g.mu.Lock()
	if g.enVuelo == nil {
		g.enVuelo = map[string]*llamada{}
	}
	if c, ok := g.enVuelo[clave]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val
	}
	c := &llamada{}
	c.wg.Add(1)
	g.enVuelo[clave] = c
	g.mu.Unlock()

	c.val = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.enVuelo, clave)
	g.mu.Unlock()
	return c.val
}

// base simula una base de datos: cada consulta tarda más cuantas más hay en vuelo (partición caliente).
type base struct {
	consultas atomic.Int64
	enVuelo   atomic.Int64
	costo     time.Duration
}

func (b *base) leer(canal string) string {
	b.consultas.Add(1)
	n := b.enVuelo.Add(1)
	defer b.enVuelo.Add(-1)
	time.Sleep(b.costo + time.Duration(n)*b.costo/50)
	return "mensajes de " + canal
}

func correr(n int, costo time.Duration, agrupar bool) (int64, time.Duration) {
	db := &base{costo: costo}
	var g grupo
	lat := make([]time.Duration, n)
	var wg sync.WaitGroup
	inicio := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-inicio
			t0 := time.Now()
			if agrupar {
				g.Do("canal-anuncios", func() string { return db.leer("canal-anuncios") })
			} else {
				db.leer("canal-anuncios")
			}
			lat[i] = time.Since(t0)
		}(i)
	}
	close(inicio)
	wg.Wait()
	sort.Slice(lat, func(a, b int) bool { return lat[a] < lat[b] })
	return db.consultas.Load(), lat[len(lat)*99/100]
}

func main() {
	n := flag.Int("n", 1000, "peticiones iguales al mismo tiempo")
	costo := flag.Duration("costo", 5*time.Millisecond, "costo base de una consulta")
	flag.Parse()
	fmt.Printf("%d peticiones al mismo canal · consulta base %v\n", *n, *costo)
	fmt.Printf("%-16s %10s %10s\n", "modo", "consultas", "p99")
	for _, agrupar := range []bool{false, true} {
		q, p99 := correr(*n, *costo, agrupar)
		modo := "sin coalescing"
		if agrupar {
			modo = "con singleflight"
		}
		fmt.Printf("%-16s %10d %10v\n", modo, q, p99.Round(time.Millisecond))
	}
}
