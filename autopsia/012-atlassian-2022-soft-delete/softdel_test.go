package softdel

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

type reloj struct{ t time.Time }

func (r *reloj) ahora() time.Time { return r.t }

func nuevo(r *reloj) *Store {
	s := New(30*24*time.Hour, 50)
	s.Ahora = r.ahora
	return s
}

func TestBorradoSuaveYRestaurar(t *testing.T) {
	r := &reloj{time.Date(2022, 4, 5, 7, 38, 0, 0, time.UTC)}
	s := nuevo(r)
	s.AgregarApp("insight-42")
	if err := s.BorrarApp("insight-42"); err != nil || s.AppActiva("insight-42") {
		t.Fatalf("la app debería quedar marcada como borrada: %v", err)
	}
	r.t = r.t.Add(13 * 24 * time.Hour)
	if err := s.RestaurarApp("insight-42"); err != nil || !s.AppActiva("insight-42") {
		t.Fatalf("dentro de la retención se debe poder restaurar: %v", err)
	}
}

func TestSitioExigeConfirmacion(t *testing.T) {
	s := nuevo(&reloj{time.Now()})
	s.AgregarSitio("site-7")
	if err := s.BorrarSitio("site-7", ""); !errors.Is(err, ErrFaltaConfirmar) || !s.SitioActivo("site-7") {
		t.Fatalf("sin confirmación no se borra: %v", err)
	}
	if err := s.BorrarSitio("site-7", "site-7"); err != nil || s.SitioActivo("site-7") {
		t.Fatalf("con confirmación sí: %v", err)
	}
}

func TestTopePorMinuto(t *testing.T) {
	r := &reloj{time.Now()}
	s := nuevo(r)
	var ids []AppID
	for i := 0; i < 883; i++ {
		id := AppID(fmt.Sprintf("app-%d", i))
		s.AgregarApp(id)
		ids = append(ids, id)
	}
	n, err := s.LoteApps(ids)
	if n != 50 || !errors.Is(err, ErrTope) {
		t.Fatalf("el lote debería frenar en 50, borró %d (%v)", n, err)
	}
	r.t = r.t.Add(time.Minute)
	if n, _ := s.LoteApps(ids[50:]); n != 50 {
		t.Fatalf("al minuto siguiente vuelven 50 borrados, hubo %d", n)
	}
}

func TestPurgaTrasRetencion(t *testing.T) {
	r := &reloj{time.Now()}
	s := nuevo(r)
	s.AgregarApp("vieja")
	_ = s.BorrarApp("vieja")
	if s.Purgar() != 0 {
		t.Fatal("no se purga antes de que venza la retención")
	}
	r.t = r.t.Add(31 * 24 * time.Hour)
	if s.Purgar() != 1 || s.RestaurarApp("vieja") == nil {
		t.Fatal("vencida la retención, se purga y ya no se puede restaurar")
	}
}
