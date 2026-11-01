package ring

import (
	"fmt"
	"testing"
)

func servers(n int) []string {
	var s []string
	for i := 1; i <= n; i++ {
		s = append(s, fmt.Sprintf("cs-%d", i))
	}
	return s
}

// Al quitar un servidor, solo se mueven los canales que eran suyos; ninguno de los demás cambia de dueño.
func TestRemoveOnlyMovesItsKeys(t *testing.T) {
	r := New(160, servers(6)...)
	before := map[string]string{}
	for i := 0; i < 20000; i++ {
		k := fmt.Sprintf("C%07d", i)
		before[k] = r.Owner(k)
	}
	r.Remove("cs-4")
	moved := 0
	for k, was := range before {
		now := r.Owner(k)
		if was != "cs-4" && now != was {
			t.Fatalf("%s cambió de %s a %s sin que su servidor saliera", k, was, now)
		}
		if now == "cs-4" {
			t.Fatalf("%s sigue en cs-4", k)
		}
		if now != was {
			moved++
		}
	}
	if f := float64(moved) / float64(len(before)); f > 0.25 {
		t.Errorf("se movió %.1f %% de los canales; esperaba cerca de 1/6", f*100)
	}
}

// Con hash % N, pasar de 6 a 5 servidores mueve la gran mayoría.
func TestModNMovesMost(t *testing.T) {
	six, five := servers(6), []string{"cs-1", "cs-2", "cs-3", "cs-5", "cs-6"}
	moved := 0
	for i := 0; i < 20000; i++ {
		k := fmt.Sprintf("C%07d", i)
		if ModN(k, six) != ModN(k, five) {
			moved++
		}
	}
	if f := float64(moved) / 20000; f < 0.75 {
		t.Errorf("hash %% N movió solo %.1f %%", f*100)
	}
}

// Con nodos virtuales el reparto queda parejo: ningún servidor se lleva más del doble que otro.
func TestBalance(t *testing.T) {
	r := New(160, servers(6)...)
	count := map[string]int{}
	for i := 0; i < 60000; i++ {
		count[r.Owner(fmt.Sprintf("C%07d", i))]++
	}
	lo, hi := 1<<30, 0
	for _, n := range count {
		lo, hi = min(lo, n), max(hi, n)
	}
	if hi > 2*lo {
		t.Errorf("reparto disparejo: %v", count)
	}
}
