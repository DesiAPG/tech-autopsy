// cache-bench: un servidor HTTP con caché en dos niveles, como Stack Overflow en 2016:
// L1 en la memoria del propio servidor, L2 compartido (el papel de Redis) y SQL Server solo si ninguno lo tiene.
package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// l1 es la caché local: un mapa en memoria con TTL, sin red de por medio.
type l1 struct {
	mu  sync.RWMutex
	m   map[string]entrada
	ttl time.Duration
}

type entrada struct {
	val    string
	expira time.Time
}

func (c *l1) get(k string) (string, bool) {
	c.mu.RLock()
	e, ok := c.m[k]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expira) {
		return "", false
	}
	return e.val, true
}

func (c *l1) set(k, v string) {
	c.mu.Lock()
	c.m[k] = entrada{v, time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

// l2 simula Redis: compartido entre servidores, rápido pero con un viaje de red en cada consulta.
type l2 struct {
	mu   sync.RWMutex
	m    map[string]string
	red  time.Duration
	hits atomic.Int64
}

func (c *l2) get(k string) (string, bool) {
	time.Sleep(c.red)
	c.mu.RLock()
	v, ok := c.m[k]
	c.mu.RUnlock()
	if ok {
		c.hits.Add(1)
	}
	return v, ok
}

func (c *l2) set(k, v string) {
	c.mu.Lock()
	c.m[k] = v
	c.mu.Unlock()
}

// sql simula la base: cada consulta tarda más cuantas más hay en vuelo.
type sql struct {
	consultas atomic.Int64
	enVuelo   atomic.Int64
	costo     time.Duration
}

func (b *sql) pregunta(id string) string {
	b.consultas.Add(1)
	n := b.enVuelo.Add(1)
	defer b.enVuelo.Add(-1)
	time.Sleep(b.costo + time.Duration(n)*b.costo/20)
	return "pregunta " + id
}

type servidor struct {
	l1     *l1
	l2     *l2
	db     *sql
	l1Hits atomic.Int64
}

func (s *servidor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if s.l1 != nil {
		if v, ok := s.l1.get(id); ok {
			s.l1Hits.Add(1)
			io.WriteString(w, v)
			return
		}
	}
	if s.l2 != nil {
		if v, ok := s.l2.get(id); ok {
			if s.l1 != nil {
				s.l1.set(id, v)
			}
			io.WriteString(w, v)
			return
		}
	}
	v := s.db.pregunta(id)
	if s.l2 != nil {
		s.l2.set(id, v)
	}
	if s.l1 != nil {
		s.l1.set(id, v)
	}
	io.WriteString(w, v)
}

type resultado struct {
	rps            float64
	p50            time.Duration
	l1, l2, sqlPct float64
}

func correr(modo string, n, par, preguntas int, costo, red time.Duration) resultado {
	s := &servidor{db: &sql{costo: costo}}
	if modo != "sin caché" {
		s.l2 = &l2{m: map[string]string{}, red: red}
	}
	if modo == "L1 + L2" {
		s.l1 = &l1{m: map[string]entrada{}, ttl: time.Minute}
	}
	ts := httptest.NewServer(s)
	defer ts.Close()
	cli := ts.Client()
	cli.Transport.(*http.Transport).MaxIdleConnsPerHost = par

	// pocas preguntas muy visitadas y muchas poco visitadas (Zipf), como cualquier sitio de preguntas
	ids := make([]string, n)
	z := rand.NewZipf(rand.New(rand.NewSource(10)), 1.1, 1, uint64(preguntas-1))
	for i := range ids {
		ids[i] = fmt.Sprint(z.Uint64())
	}
	lat := make([]time.Duration, n)
	var sig atomic.Int64
	var wg sync.WaitGroup
	t0 := time.Now()
	for g := 0; g < par; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(sig.Add(1) - 1)
				if i >= n {
					return
				}
				ti := time.Now()
				res, err := cli.Get(ts.URL + "/q?id=" + ids[i])
				if err != nil {
					panic(err)
				}
				io.Copy(io.Discard, res.Body)
				res.Body.Close()
				lat[i] = time.Since(ti)
			}
		}()
	}
	wg.Wait()
	total := time.Since(t0)
	sort.Slice(lat, func(a, b int) bool { return lat[a] < lat[b] })
	pct := func(x int64) float64 { return 100 * float64(x) / float64(n) }
	r := resultado{rps: float64(n) / total.Seconds(), p50: lat[n/2], sqlPct: pct(s.db.consultas.Load()), l1: pct(s.l1Hits.Load())}
	if s.l2 != nil {
		r.l2 = pct(s.l2.hits.Load())
	}
	return r
}

func main() {
	n := flag.Int("n", 20000, "peticiones en total")
	par := flag.Int("par", 64, "peticiones en paralelo")
	preguntas := flag.Int("preguntas", 5000, "preguntas distintas")
	costo := flag.Duration("sql", 5*time.Millisecond, "costo base de una consulta a SQL")
	red := flag.Duration("red", 300*time.Microsecond, "viaje de red a L2 (Redis)")
	flag.Parse()
	fmt.Printf("1 instancia · %d peticiones · %d en paralelo\n", *n, *par)
	fmt.Printf("%-10s %7s %7s %5s %5s %5s\n", "modo", "req/s", "p50", "L1", "L2", "SQL")
	for _, modo := range []string{"sin caché", "solo L2", "L1 + L2"} {
		r := correr(modo, *n, *par, *preguntas, *costo, *red)
		fmt.Printf("%-10s %7.0f %7v %4.0f%% %4.0f%% %4.0f%%\n", modo, r.rps, r.p50.Round(10*time.Microsecond), r.l1, r.l2, r.sqlPct)
	}
}
