// Package fanout muestra las dos lecciones de concurrencia del rediseño de Twitter (2010–2013):
//
//  1. En Rails, cada proceso atendía una petición a la vez, así que una página que compone varios servicios
//     suma sus latencias. Con futures (aquí goroutines) se piden en paralelo y la página tarda lo del más lento.
//  2. Al partir el monolito, los clientes reintentaban sin parar contra servicios lentos ("thundering herds").
//     Finagle resolvió eso con una sola librería de red; aquí, un presupuesto de reintentos acota la estampida.
package fanout

import (
	"context"
	"sync"
	"time"
)

// Call es una llamada a un servicio (tuits, timelines, usuarios…).
type Call func(ctx context.Context) error

// Sequential llama a los servicios uno tras otro, como un proceso de un solo hilo: la latencia se suma.
func Sequential(ctx context.Context, calls ...Call) (time.Duration, error) {
	t0 := time.Now()
	for _, c := range calls {
		if err := c(ctx); err != nil {
			return time.Since(t0), err
		}
	}
	return time.Since(t0), nil
}

// Concurrent lanza todas las llamadas a la vez y espera a que terminen: la latencia es la del más lento.
func Concurrent(ctx context.Context, calls ...Call) (time.Duration, error) {
	t0 := time.Now()
	var wg sync.WaitGroup
	errs := make([]error, len(calls))
	for i, c := range calls {
		wg.Add(1)
		go func(i int, c Call) {
			defer wg.Done()
			errs[i] = c(ctx)
		}(i, c)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return time.Since(t0), err
		}
	}
	return time.Since(t0), nil
}

// Service simula un servicio con una latencia fija usando el reloj real; sirve para la demo.
func Service(latency time.Duration) Call {
	return func(ctx context.Context) error {
		select {
		case <-time.After(latency):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
