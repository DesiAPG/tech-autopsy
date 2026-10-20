// Package poolbench mide cuánto cuesta no reutilizar conexiones: el mismo tráfico contra un
// origen HTTPS local con tres estrategias de pool (ninguno, uno por worker, uno compartido).
package poolbench

import (
	"crypto/tls"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Mode es la estrategia de conexiones del proxy.
type Mode int

const (
	NoReuse   Mode = iota // una conexión nueva (TCP + TLS) por petición
	PerWorker             // un pool por worker, como los procesos de NGINX
	Shared                // un solo pool para todos, como los hilos de Pingora
)

func (m Mode) String() string {
	return [...]string{"sin reutilizar", "pool por worker", "pool compartido"}[m]
}

// Config describe la carga.
type Config struct {
	Requests    int // peticiones totales
	Concurrency int // peticiones en vuelo a la vez
	Workers     int // workers (pools) en el modo PerWorker
	Seed        uint64
}

// Result resume una ejecución.
type Result struct {
	Mode       Mode
	Requests   int
	Handshakes int64 // conexiones TLS nuevas que aceptó el origen
	P50, P95   time.Duration
}

// Reuse es el porcentaje de peticiones que usaron una conexión ya abierta.
func (r Result) Reuse() float64 {
	return 100 * (1 - float64(r.Handshakes)/float64(r.Requests))
}

// Run levanta un origen HTTPS local y le envía cfg.Requests peticiones con la estrategia m.
func Run(m Mode, cfg Config) (Result, error) {
	var handshakes atomic.Int64
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "ok")
	}))
	srv.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			handshakes.Add(1)
		}
	}
	srv.StartTLS()
	defer srv.Close()

	newTransport := func() *http.Transport {
		t := srv.Client().Transport.(*http.Transport).Clone()
		t.TLSClientConfig = &tls.Config{RootCAs: srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs}
		t.MaxIdleConnsPerHost = cfg.Concurrency
		t.DisableKeepAlives = m == NoReuse
		return t
	}
	pools := 1
	if m == PerWorker {
		pools = cfg.Workers
	}
	clients := make([]*http.Client, pools)
	for i := range clients {
		clients[i] = &http.Client{Transport: newTransport()}
	}
	defer func() {
		for _, c := range clients {
			c.Transport.(*http.Transport).CloseIdleConnections()
		}
	}()

	// cada petición cae en un worker al azar (determinista por Seed)
	rng := rand.New(rand.NewPCG(cfg.Seed, 4))
	jobs := make(chan int, cfg.Requests)
	for range cfg.Requests {
		jobs <- rng.IntN(pools)
	}
	close(jobs)

	lat := make([]time.Duration, 0, cfg.Requests)
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error
	for range cfg.Concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				t0 := time.Now()
				resp, err := clients[p].Get(srv.URL)
				if err == nil {
					_, err = io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
				d := time.Since(t0)
				mu.Lock()
				if err != nil && firstErr == nil {
					firstErr = err
				}
				lat = append(lat, d)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if firstErr != nil {
		return Result{}, fmt.Errorf("%s: %w", m, firstErr)
	}
	sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
	return Result{
		Mode: m, Requests: cfg.Requests, Handshakes: handshakes.Load(),
		P50: lat[len(lat)/2], P95: lat[len(lat)*95/100],
	}, nil
}
