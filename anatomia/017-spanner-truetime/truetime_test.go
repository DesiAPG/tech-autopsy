package truetime

import (
	"math/rand"
	"testing"
	"time"
)

const ms = time.Millisecond

// Con el reloj del pago atrasado 3 ms, las marcas ingenuas ponen el pago antes que el depósito y una lectura ve −50.
func TestNaiveBankPenalty(t *testing.T) {
	a := Node{Name: "a", Skew: 0, Eps: 4 * ms}
	b := Node{Name: "b", Skew: -3 * ms, Eps: 4 * ms}
	dep := CommitNaive(a, 100*ms)
	pay := CommitNaive(b, 101*ms) // empieza 1 ms después de que el depósito terminó
	ops := []Op{{"depósito", "ahorros", 200, dep}, {"pago", "corriente", -150, pay}}
	if Ordered(dep, 101*ms, pay) {
		t.Fatalf("se esperaba el orden invertido: depósito %v, pago %v", dep.TS, pay.TS)
	}
	if !Penalty(ops) {
		t.Fatal("se esperaba una lectura con saldo negativo")
	}
}

// Con TrueTime y commit wait, el mismo caso respeta el orden y nunca se cobra multa.
func TestCommitWaitBank(t *testing.T) {
	a := Node{Name: "a", Skew: 0, Eps: 4 * ms}
	b := Node{Name: "b", Skew: -3 * ms, Eps: 4 * ms}
	dep := CommitWait(a, 100*ms, 100*time.Microsecond)
	pay := CommitWait(b, dep.Done+1*ms, 100*time.Microsecond)
	ops := []Op{{"depósito", "ahorros", 200, dep}, {"pago", "corriente", -150, pay}}
	if !Ordered(dep, dep.Done+1*ms, pay) || Penalty(ops) {
		t.Fatalf("orden roto: depósito %v, pago %v", dep.TS, pay.TS)
	}
}

// Propiedad: con relojes que respetan su cota (|skew| <= ε), commit wait nunca invierte el orden real.
// Sin esperar, con los mismos relojes, sí pasa.
func TestExternalConsistency(t *testing.T) {
	r := rand.New(rand.NewSource(17))
	inverted := 0
	for i := 0; i < 5000; i++ {
		eps := time.Duration(1+r.Intn(7)) * ms
		skew := func() time.Duration { return time.Duration(r.Int63n(int64(2*eps+1))) - eps }
		a, b := Node{"a", skew(), eps}, Node{"b", skew(), eps}
		start := time.Duration(r.Intn(1000)) * ms
		gap := time.Duration(r.Intn(3000)) * time.Microsecond

		c1 := CommitWait(a, start, 50*time.Microsecond)
		c2 := CommitWait(b, c1.Done+gap+time.Microsecond, 50*time.Microsecond)
		if !Ordered(c1, c1.Done+gap+time.Microsecond, c2) {
			t.Fatalf("commit wait invirtió el orden: %+v %+v", c1, c2)
		}
		if c1.Wait < 2*eps-time.Millisecond/10 {
			t.Fatalf("la espera (%v) debería ser al menos ~2ε (%v)", c1.Wait, 2*eps)
		}
		n1 := CommitNaive(a, start)
		n2 := CommitNaive(b, start+gap+time.Microsecond)
		if !Ordered(n1, start+gap+time.Microsecond, n2) {
			inverted++
		}
	}
	if inverted == 0 {
		t.Fatal("se esperaba que las marcas ingenuas invirtieran el orden alguna vez")
	}
}
