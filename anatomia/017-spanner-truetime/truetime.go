// Package truetime reproduce, en tiempo simulado, la idea de TrueTime y commit wait de Spanner (OSDI 2012):
// cada máquina tiene un reloj que se desvía, así que TT.Now() no devuelve una hora sino un intervalo
// [Earliest, Latest] que contiene la hora real. Una escritura toma s = TT.Now().Latest y espera hasta que
// TT.After(s) sea cierto antes de confirmarse. Así, si T1 termina antes de que empiece T2, s1 < s2 en cualquier máquina.
package truetime

import "time"

// Interval es lo que devuelve TT.Now(): la hora real está garantizada adentro.
type Interval struct{ Earliest, Latest time.Duration }

// Node es una máquina con su reloj local: Skew es cuánto se desvía de la hora real y Eps la cota que anuncia
// (en Spanner, ε sale de los maestros de tiempo y de la deriva máxima; suele estar entre 1 y 7 ms).
// La garantía solo vale si |Skew| <= Eps.
type Node struct {
	Name string
	Skew time.Duration
	Eps  time.Duration
}

// Local es la hora que marca el reloj de la máquina cuando la hora real es real.
func (n Node) Local(real time.Duration) time.Duration { return real + n.Skew }

// Now es TT.now(): un intervalo alrededor de la hora local, de ancho 2ε.
func (n Node) Now(real time.Duration) Interval {
	l := n.Local(real)
	return Interval{Earliest: l - n.Eps, Latest: l + n.Eps}
}

// After es TT.after(s): verdadero si s ya pasó con seguridad.
func (n Node) After(real, s time.Duration) bool { return n.Now(real).Earliest > s }

// Commit es el resultado de confirmar una escritura en una máquina.
type Commit struct {
	TS   time.Duration // marca de tiempo de la escritura
	Done time.Duration // hora real en que la escritura queda visible
	Wait time.Duration // cuánto esperó a propósito (commit wait)
}

// CommitNaive marca la escritura con el reloj local, sin esperar: lo que hace una base que confía en un solo reloj.
func CommitNaive(n Node, start time.Duration) Commit {
	return Commit{TS: n.Local(start), Done: start}
}

// CommitWait es la regla de Spanner: s = TT.now().latest y espera hasta TT.after(s).
// La espera se avanza en pasos de step de tiempo simulado (no duerme de verdad).
func CommitWait(n Node, start, step time.Duration) Commit {
	s := n.Now(start).Latest
	t := start
	for !n.After(t, s) {
		t += step
	}
	return Commit{TS: s, Done: t, Wait: t - start}
}

// Ordered dice si las marcas respetan el orden real: si a terminó antes de que empezara b, a.TS < b.TS.
func Ordered(a Commit, bStart time.Duration, b Commit) bool {
	if a.Done >= bStart { // no hubo un "antes" claro: no hay nada que exigir
		return true
	}
	return a.TS < b.TS
}
