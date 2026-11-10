package fanout

import (
	"context"
	"testing"
	"time"
)

// En secuencia la página tarda la suma; en paralelo, lo del servicio más lento.
func TestSequentialAddsConcurrentTakesMax(t *testing.T) {
	ms := time.Millisecond
	calls := []Call{Service(20 * ms), Service(30 * ms), Service(40 * ms)}
	seq, err := Sequential(context.Background(), calls...)
	if err != nil || seq < 90*ms {
		t.Fatalf("secuencial: %v (err %v), se esperaba >= 90ms", seq, err)
	}
	con, err := Concurrent(context.Background(), calls...)
	if err != nil || con < 40*ms || con > 80*ms {
		t.Fatalf("concurrente: %v (err %v), se esperaba ~40ms", con, err)
	}
}

// Sin presupuesto, un servicio lento recibe una estampida de reintentos; con presupuesto, ~20 % extra como mucho.
func TestBudgetCapsRetries(t *testing.T) {
	free := Load(1000, 3, 0.9, nil)
	b := &Budget{Ratio: 0.2, MinRetries: 10}
	capped := Load(1000, 3, 0.9, b)
	if free < 3000 {
		t.Fatalf("sin presupuesto se esperaban más de 3000 intentos, hubo %d", free)
	}
	if capped > 1000+200+10 {
		t.Fatalf("con presupuesto se esperaban <= 1210 intentos, hubo %d", capped)
	}
}
