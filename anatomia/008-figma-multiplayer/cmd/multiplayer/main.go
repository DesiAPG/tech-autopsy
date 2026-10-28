// multiplayer: demo del modelo de Figma con un servidor y dos clientes reales por WebSocket.
//
//	go run ./anatomia/008-figma-multiplayer/cmd/multiplayer
package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	mp "github.com/DesiAPG/tech-autopsy/anatomia/008-figma-multiplayer"
)

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fail(err)
	}
	srv := mp.NewServer()
	go http.Serve(ln, srv)
	url := "ws://" + ln.Addr().String() + "/"
	ana, beto := mp.NewClient(1, "ANA", url), mp.NewClient(2, "BETO", url)
	for _, c := range []*mp.Client{ana, beto} {
		if err := c.Connect(); err != nil {
			fail(err)
		}
	}
	settle := func(cs ...*mp.Client) {
		if !mp.Settle(srv, 3*time.Second, cs...) {
			fail(fmt.Errorf("los clientes no convergieron"))
		}
	}
	fmt.Printf("servidor %s · clientes ANA (id 1) y BETO (id 2)\n", url)

	paso("1 · ANA crea un rectángulo: el ID lleva su prefijo de cliente")
	r := ana.NewObjectID()
	ana.Set(r, "pos", mp.Between("", ""))
	ana.Set(r, "x", "120")
	ana.Set(r, "color", "blanco")
	settle(ana, beto)
	fmt.Print(srv.Doc())

	paso("2 · a la vez, propiedades distintas: ANA mueve (x), BETO pinta (color)")
	both(func() { ana.Set(r, "x", "340") }, func() { beto.Set(r, "color", "azul") })
	settle(ana, beto)
	fmt.Print(srv.Doc())
	fmt.Println("  ✓ sin conflicto: se quedan los dos cambios")

	paso("3 · a la vez, la MISMA propiedad: ANA naranja, BETO blanco")
	both(func() { ana.Set(r, "color", "naranja") }, func() { beto.Set(r, "color", "blanco") })
	settle(ana, beto)
	log := srv.Log(r, "color")
	for i, m := range log[len(log)-2:] {
		fmt.Printf("  llegó %d · %-4s color = %s\n", i+1, name(m.From), m.Val)
	}
	fmt.Printf("  ✓ gana el último que llegó al servidor: color = %s (en los 3)\n", srv.Doc().Get(r, "color"))

	paso("4 · fractional indexing: capas nuevas entre dos, sin renumerar")
	p1 := srv.Doc().Get(r, "pos")
	r2 := ana.NewObjectID()
	ana.Set(r2, "pos", mp.Between(p1, ""))
	r3 := beto.NewObjectID()
	beto.Set(r3, "pos", mp.Between(p1, mp.Between(p1, "")))
	settle(ana, beto)
	fmt.Print(srv.Doc())

	paso("5 · ANA se queda sin internet y sigue editando")
	ana.Disconnect()
	ana.Set(r, "x", "500")
	ana.Set(r, "color", "verde")
	beto.Set(r, "color", "azul")
	beto.Set(r, "y", "80")
	settle(beto)
	fmt.Printf("  ANA offline: x=500, color=verde · BETO online: color=azul, y=80\n")
	fmt.Println("  ANA vuelve: baja el documento fresco y reaplica encima solo sus cambios")
	if err := ana.Connect(); err != nil {
		fail(err)
	}
	settle(ana, beto)
	d := srv.Doc()
	fmt.Print(d)
	fmt.Printf("  ✓ y=%s de BETO se conserva (ANA no la tocó)\n", d.Get(r, "y"))
	fmt.Printf("  × color=%s: ANA pisó el azul de BETO (los dos tocaron la misma propiedad)\n", d.Get(r, "color"))

	paso("final")
	if ana.Doc().Equal(d) && beto.Doc().Equal(d) {
		fmt.Println("  ✓ servidor, ANA y BETO tienen exactamente el mismo documento")
	} else {
		fail(fmt.Errorf("no convergieron"))
	}
}

func both(a, b func()) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); a() }()
	go func() { defer wg.Done(); b() }()
	wg.Wait()
}

func name(id int) string { return map[int]string{1: "ANA", 2: "BETO"}[id] }

func paso(s string) { fmt.Printf("\n%s\n", s) }

func fail(err error) { fmt.Fprintln(os.Stderr, "error:", err); os.Exit(1) }
