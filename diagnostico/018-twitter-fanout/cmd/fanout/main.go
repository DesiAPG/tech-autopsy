// fanout: latencia que se suma (un proceso, una petición) vs servicios en paralelo, y la estampida de reintentos
// con y sin presupuesto.
//
//	go run ./diagnostico/018-twitter-fanout/cmd/fanout
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/DesiAPG/tech-autopsy/diagnostico/018-twitter-fanout"
)

func main() {
	ms := time.Millisecond
	calls := []fanout.Call{fanout.Service(30 * ms), fanout.Service(45 * ms), fanout.Service(60 * ms)}
	fmt.Println("una página = tuits (30 ms) + timeline (45 ms) + usuario (60 ms)")
	seq, _ := fanout.Sequential(context.Background(), calls...)
	con, _ := fanout.Concurrent(context.Background(), calls...)
	fmt.Printf("  un hilo, en secuencia:  %v  (se suma)\n", seq.Round(ms))
	fmt.Printf("  futures, en paralelo:   %v  (el más lento)\n\n", con.Round(ms))

	fmt.Println("1000 peticiones a un servicio lento (90 % de timeouts), hasta 3 reintentos por cliente")
	free := fanout.Load(1000, 3, 0.9, nil)
	capped := fanout.Load(1000, 3, 0.9, &fanout.Budget{Ratio: 0.2, MinRetries: 10})
	fmt.Printf("  cada cliente reintenta por su cuenta: %d intentos  ✗ estampida\n", free)
	fmt.Printf("  presupuesto de reintentos (20 %%):     %d intentos  ✓\n", capped)
}
