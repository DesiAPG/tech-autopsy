// Package qslog es una mini réplica del log de Quicksilver: un único escritor numera cada
// cambio con una secuencia monótona (+1), los agrupa en lotes y los envía a réplicas por una
// red que pierde mensajes. Cada réplica detecta los huecos y pide el reenvío. Para comparar,
// también replica por timestamps, que pierde entradas sin que nadie se entere.
package qslog

import (
	"hash/fnv"
	"math/rand/v2"
	"sort"
)

// Entry es un cambio de configuración.
type Entry struct {
	Seq   uint64 // secuencia del log: exactamente +1 respecto a la anterior
	TS    int64  // reloj del escritor que la originó (lo que usa la replicación por timestamps)
	Key   string
	Value string
}

// Config describe la simulación.
type Config struct {
	Changes  int     // cambios totales
	Replicas int     // réplicas
	Loss     float64 // probabilidad de perder cada mensaje (0–1)
	Batch    int     // cambios por lote (una "ventana de 500 ms")
	Skew     int64   // desfase máximo entre relojes de los escritores, en ms
	Seed     uint64
}

// Result resume una estrategia.
type Result struct {
	Gaps      int  // huecos detectados (solo secuencia)
	Resent    int  // entradas reenviadas
	Lost      int  // entradas que alguna réplica nunca aplicó
	Identical bool // todas las réplicas terminan con el mismo estado que el líder
}

func digest(m map[string]string) uint64 {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := fnv.New64a()
	for _, k := range keys {
		h.Write([]byte(k + "=" + m[k] + ";"))
	}
	return h.Sum64()
}

// Log genera los cambios: varios escritores con relojes desfasados, ordenados por el líder.
func Log(cfg Config) []Entry {
	rng := rand.New(rand.NewPCG(cfg.Seed, 1))
	skew := []int64{0, cfg.Skew, -cfg.Skew / 2} // reloj de cada escritor
	log := make([]Entry, cfg.Changes)
	for i := range log {
		w := rng.IntN(len(skew))
		log[i] = Entry{
			Seq:   uint64(i + 1),
			TS:    int64(i)*20 + skew[w], // 20 ms entre cambios, más el desfase del reloj
			Key:   "k" + itoa(rng.IntN(cfg.Changes/10+1)),
			Value: "v" + itoa(i),
		}
	}
	return log
}

// Sequence replica con secuencia: un hueco se detecta y se pide de nuevo.
func Sequence(cfg Config, log []Entry) Result {
	rng := rand.New(rand.NewPCG(cfg.Seed, 2))
	leader := apply(log)
	var res Result
	res.Identical = true
	for range cfg.Replicas {
		state := map[string]string{}
		var next uint64 = 1
		pending := map[uint64]Entry{}
		for b := 0; b < len(log); b += cfg.Batch {
			for _, e := range log[b:min(b+cfg.Batch, len(log))] {
				if rng.Float64() < cfg.Loss {
					continue // la red se lo comió
				}
				pending[e.Seq] = e
			}
			// aplica en orden; si falta una secuencia, es un hueco: se pide al de arriba
			for next <= uint64(min(b+cfg.Batch, len(log))) {
				e, ok := pending[next]
				if !ok {
					res.Gaps++
					res.Resent++
					e = log[next-1] // reenvío
				}
				delete(pending, next)
				state[e.Key] = e.Value
				next++
			}
		}
		if digest(state) != leader {
			res.Identical = false
		}
	}
	return res
}

// Timestamps replica por reloj: aplica solo lo más nuevo que lo último que vio.
// Una entrada perdida o con un reloj atrasado se descarta, y nadie avisa.
func Timestamps(cfg Config, log []Entry) Result {
	rng := rand.New(rand.NewPCG(cfg.Seed, 2))
	leader := apply(log)
	var res Result
	res.Identical = true
	for range cfg.Replicas {
		state := map[string]string{}
		var last int64 = -1 << 62
		for _, e := range log {
			if rng.Float64() < cfg.Loss || e.TS <= last {
				res.Lost++
				continue
			}
			last = e.TS
			state[e.Key] = e.Value
		}
		if digest(state) != leader {
			res.Identical = false
		}
	}
	return res
}

func apply(log []Entry) uint64 {
	m := map[string]string{}
	for _, e := range log {
		m[e.Key] = e.Value
	}
	return digest(m)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for ; n > 0; n /= 10 {
		i--
		b[i] = byte('0' + n%10)
	}
	return string(b[i:])
}
