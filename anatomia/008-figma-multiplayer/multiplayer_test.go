package multiplayer

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBetween(t *testing.T) {
	cases := [][3]string{{"", "", "5"}, {"3", "4", "35"}, {"35", "4", "37"}, {"3", "31", "305"}, {"", "1", "05"}, {"9", "", "95"}}
	for _, c := range cases {
		if got := Between(c[0], c[1]); got != c[2] {
			t.Errorf("Between(%q, %q) = %q, quería %q", c[0], c[1], got, c[2])
		}
	}
	// insertar 200 veces justo encima del primero: siempre queda estrictamente en medio
	lo, hi := "", "5"
	for i := 0; i < 200; i++ {
		m := Between(lo, hi)
		if !(m > lo && m < hi) {
			t.Fatalf("%q no está entre %q y %q", m, lo, hi)
		}
		hi = m
	}
}

func setup(t *testing.T, n int) (*Server, []*Client) {
	s := NewServer()
	hs := httptest.NewServer(s)
	t.Cleanup(hs.Close)
	url := "ws" + strings.TrimPrefix(hs.URL, "http") + "/"
	cs := make([]*Client, n)
	for i := range cs {
		cs[i] = NewClient(i+1, fmt.Sprint("C", i+1), url)
		if err := cs[i].Connect(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(cs[i].Disconnect)
	}
	return s, cs
}

func TestPropiedadesDistintasNoChocan(t *testing.T) {
	s, cs := setup(t, 2)
	cs[0].Set("1:1", "x", "340")
	cs[1].Set("1:1", "color", "azul")
	if !Settle(s, 2*time.Second, cs...) {
		t.Fatal("no convergen")
	}
	if d := s.Doc(); d.Get("1:1", "x") != "340" || d.Get("1:1", "color") != "azul" {
		t.Fatalf("se perdió un cambio:\n%s", d)
	}
}

func TestMismaPropiedadGanaElUltimo(t *testing.T) {
	s, cs := setup(t, 4)
	var wg sync.WaitGroup
	for i, c := range cs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				c.Set("1:1", "color", fmt.Sprintf("c%d-%d", i, j))
			}
		}()
	}
	wg.Wait()
	if !Settle(s, 5*time.Second, cs...) {
		t.Fatal("no convergen")
	}
	log := s.Log("1:1", "color")
	if got, want := s.Doc().Get("1:1", "color"), log[len(log)-1].Val; got != want {
		t.Fatalf("ganó %q, pero el último en llegar fue %q", got, want)
	}
}

func TestOfflineReaplicaSobreLoMasNuevo(t *testing.T) {
	s, cs := setup(t, 2)
	ana, beto := cs[0], cs[1]
	ana.Set("1:1", "color", "blanco")
	Settle(s, 2*time.Second, cs...)
	ana.Disconnect()
	ana.Set("1:1", "x", "500")       // solo ANA toca x
	ana.Set("1:1", "color", "verde") // los dos tocan color
	beto.Set("1:1", "color", "azul")
	beto.Set("1:1", "y", "80") // solo BETO toca y
	Settle(s, 2*time.Second, beto)
	if err := ana.Connect(); err != nil {
		t.Fatal(err)
	}
	if !Settle(s, 2*time.Second, cs...) {
		t.Fatal("no convergen")
	}
	d := s.Doc()
	if d.Get("1:1", "y") != "80" || d.Get("1:1", "x") != "500" || d.Get("1:1", "color") != "verde" {
		t.Fatalf("estado inesperado:\n%s", d)
	}
}
