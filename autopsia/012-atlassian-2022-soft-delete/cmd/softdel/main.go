// softdel: la demo del caso 012 (Atlassian, abril de 2022) con IDs tipados, borrado suave y tope por minuto.
package main

import (
	"fmt"
	"os"
	"time"

	softdel "github.com/DesiAPG/tech-autopsy/autopsia/012-atlassian-2022-soft-delete"
)

func check(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}

func main() {
	if len(os.Args) < 2 || os.Args[1] != "demo" {
		fmt.Println("uso: softdel demo")
		os.Exit(2)
	}
	s := softdel.New(30*24*time.Hour, 50)
	s.AgregarApp("insight-42")
	s.AgregarSitio("site-7")
	fmt.Println("$ softdel demo")

	err := s.BorrarApp("insight-42")
	fmt.Printf("app insight-42: borrado suave %s\n", check(err == nil))

	// Pasar el ID del sitio donde iba el de la app ni siquiera compila: s.BorrarApp(softdel.SiteID("site-7")).
	// Y borrar el sitio entero exige repetir su ID como confirmación.
	if err := s.BorrarSitio("site-7", ""); err != nil {
		fmt.Printf("site-7: %v %s\n", err, check(false))
	}

	var lote []softdel.AppID
	for i := 0; i < 883; i++ {
		id := softdel.AppID(fmt.Sprintf("app-%03d", i))
		s.AgregarApp(id)
		lote = append(lote, id)
	}
	if n, err := s.LoteApps(lote); err != nil {
		fmt.Printf("lote de %d: frenado a %d/min %s\n", len(lote), s.TopeMin, check(n == len(lote)))
	}

	err = s.RestaurarApp("insight-42")
	fmt.Printf("restaurar insight-42 %s\n", check(err == nil && s.AppActiva("insight-42")))
}
