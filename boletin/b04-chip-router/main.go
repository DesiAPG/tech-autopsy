// chip-router: reparte trabajos entre flotas de aceleradores distintas (precio, velocidad y capacidad propias)
// y lo compara con mandar todo a una sola flota. Valores simulados: no son precios reales de ningún proveedor.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"sort"
)

type flota struct {
	nombre    string
	precio    float64 // costo por unidad de trabajo
	velocidad float64 // unidades de trabajo por tick y por slot
	slots     int     // trabajos simultáneos
	libre     []float64
}

type trabajo struct {
	llega  float64
	tamano float64
}

// asignar elige, para cada trabajo, la flota que minimiza costo + espera ponderada.
func correr(flotas []*flota, ts []trabajo, pesoEspera float64) (costo, esperaMedia, esperaP95 float64) {
	for _, f := range flotas {
		f.libre = make([]float64, f.slots)
	}
	esperas := make([]float64, len(ts))
	for i, t := range ts {
		mejor, mejorSlot, mejorScore := -1, 0, 0.0
		for fi, f := range flotas {
			s := 0
			for j := range f.libre {
				if f.libre[j] < f.libre[s] {
					s = j
				}
			}
			inicio := max(t.llega, f.libre[s])
			score := f.precio*t.tamano + pesoEspera*(inicio-t.llega)
			if mejor < 0 || score < mejorScore {
				mejor, mejorSlot, mejorScore = fi, s, score
			}
		}
		f := flotas[mejor]
		inicio := max(t.llega, f.libre[mejorSlot])
		f.libre[mejorSlot] = inicio + t.tamano/f.velocidad
		costo += f.precio * t.tamano
		esperas[i] = inicio - t.llega
	}
	sort.Float64s(esperas)
	for _, e := range esperas {
		esperaMedia += e
	}
	return costo, esperaMedia / float64(len(esperas)), esperas[len(esperas)*95/100]
}

func nuevas() []*flota {
	return []*flota{
		{nombre: "flota-1", precio: 1.00, velocidad: 1.0, slots: 40},
		{nombre: "flota-2", precio: 0.80, velocidad: 0.8, slots: 30},
		{nombre: "flota-3", precio: 1.30, velocidad: 1.4, slots: 20},
		{nombre: "flota-4", precio: 0.90, velocidad: 0.9, slots: 25},
	}
}

func main() {
	n := flag.Int("n", 20000, "trabajos")
	ritmo := flag.Float64("ritmo", 9, "trabajos que llegan por tick")
	flag.Parse()

	r := rand.New(rand.NewSource(4))
	ts := make([]trabajo, *n)
	for i := range ts {
		ts[i] = trabajo{llega: float64(i) / *ritmo, tamano: 2 + r.ExpFloat64()*4}
	}
	fmt.Printf("%d trabajos · %.0f por tick · valores simulados\n", *n, *ritmo)
	fmt.Printf("%-16s %9s %8s %8s\n", "modo", "costo", "espera", "p95")
	todas := nuevas()
	solo := []*flota{todas[0]}
	for _, c := range []struct {
		modo   string
		flotas []*flota
	}{{"1 sola flota", solo}, {"4 flotas", nuevas()}} {
		costo, media, p95 := correr(c.flotas, ts, 2)
		fmt.Printf("%-16s %9.0f %8.1f %8.1f\n", c.modo, costo, media, p95)
	}
}
