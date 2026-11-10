package truetime

import (
	"sort"
	"time"
)

// Op es un movimiento del ejemplo del banco de la documentación de Spanner (Google Cloud):
// 50 USD en ahorros y 50 en la corriente; T1 deposita 200 en ahorros y después T2 debita 150 de la corriente.
type Op struct {
	Name    string
	Account string
	Amount  int
	C       Commit
}

// Snapshot es una lectura a la marca de tiempo ts: ve los movimientos con TS <= ts.
func Snapshot(ops []Op, ts time.Duration) (balances map[string]int, total int) {
	balances = map[string]int{"ahorros": 50, "corriente": 50}
	sorted := append([]Op(nil), ops...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].C.TS < sorted[j].C.TS })
	for _, o := range sorted {
		if o.C.TS <= ts {
			balances[o.Account] += o.Amount
		}
	}
	return balances, balances["ahorros"] + balances["corriente"]
}

// Penalty recorre todas las lecturas posibles (una justo después de cada movimiento) y dice si en alguna
// el total quedó en negativo: según el ejemplo de Google, eso cobra multa.
func Penalty(ops []Op) bool {
	for _, o := range ops {
		if _, total := Snapshot(ops, o.C.TS); total < 0 {
			return true
		}
	}
	return false
}
