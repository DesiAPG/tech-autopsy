// Package softdel: un borrado que no puede repetir el 5 de abril de 2022 de Atlassian.
// IDs con tipo (AppID y SiteID no se mezclan: pasar uno por el otro no compila), borrado suave
// con retención y restauración, confirmación explícita para borrar un sitio entero y un tope
// de borrados por minuto para los lotes.
package softdel

import (
	"errors"
	"time"
)

type AppID string
type SiteID string

var (
	ErrNoExiste        = errors.New("no existe")
	ErrFaltaConfirmar  = errors.New("falta confirmar el tipo")
	ErrTope            = errors.New("tope de borrados por minuto")
	ErrRetencionVencio = errors.New("la retención ya venció")
)

type registro struct {
	borradoEn time.Time // cero = activo
}

type Store struct {
	Retencion time.Duration // cuánto vive un borrado suave antes de purgarse
	TopeMin   int           // máximo de borrados por minuto
	Ahora     func() time.Time

	apps      map[AppID]*registro
	sitios    map[SiteID]*registro
	ventana   time.Time
	enVentana int
}

func New(retencion time.Duration, topeMin int) *Store {
	return &Store{Retencion: retencion, TopeMin: topeMin, Ahora: time.Now, apps: map[AppID]*registro{}, sitios: map[SiteID]*registro{}}
}

func (s *Store) AgregarApp(id AppID)     { s.apps[id] = &registro{} }
func (s *Store) AgregarSitio(id SiteID)  { s.sitios[id] = &registro{} }
func (s *Store) AppActiva(id AppID) bool { r, ok := s.apps[id]; return ok && r.borradoEn.IsZero() }
func (s *Store) SitioActivo(id SiteID) bool {
	r, ok := s.sitios[id]
	return ok && r.borradoEn.IsZero()
}

// cupo aplica el tope de borrados por minuto.
func (s *Store) cupo() error {
	now := s.Ahora()
	if now.Sub(s.ventana) >= time.Minute {
		s.ventana, s.enVentana = now, 0
	}
	if s.enVentana >= s.TopeMin {
		return ErrTope
	}
	s.enVentana++
	return nil
}

// BorrarApp marca la app como borrada; se puede restaurar mientras dure la retención.
func (s *Store) BorrarApp(id AppID) error {
	r, ok := s.apps[id]
	if !ok {
		return ErrNoExiste
	}
	if err := s.cupo(); err != nil {
		return err
	}
	r.borradoEn = s.Ahora()
	return nil
}

// BorrarSitio exige repetir el ID del sitio como confirmación: nadie borra un sitio entero por accidente.
func (s *Store) BorrarSitio(id, confirmacion SiteID) error {
	r, ok := s.sitios[id]
	if !ok {
		return ErrNoExiste
	}
	if confirmacion != id {
		return ErrFaltaConfirmar
	}
	if err := s.cupo(); err != nil {
		return err
	}
	r.borradoEn = s.Ahora()
	return nil
}

// RestaurarApp deshace un borrado suave dentro de la retención.
func (s *Store) RestaurarApp(id AppID) error {
	r, ok := s.apps[id]
	if !ok {
		return ErrNoExiste
	}
	if !r.borradoEn.IsZero() && s.Ahora().Sub(r.borradoEn) > s.Retencion {
		return ErrRetencionVencio
	}
	r.borradoEn = time.Time{}
	return nil
}

// LoteApps borra apps una por una y se detiene en el tope; devuelve cuántas borró.
func (s *Store) LoteApps(ids []AppID) (int, error) {
	for i, id := range ids {
		if err := s.BorrarApp(id); err != nil {
			return i, err
		}
	}
	return len(ids), nil
}

// Purgar elimina de verdad lo que lleva más que la retención marcado como borrado.
func (s *Store) Purgar() int {
	n, now := 0, s.Ahora()
	for id, r := range s.apps {
		if !r.borradoEn.IsZero() && now.Sub(r.borradoEn) > s.Retencion {
			delete(s.apps, id)
			n++
		}
	}
	for id, r := range s.sitios {
		if !r.borradoEn.IsZero() && now.Sub(r.borradoEn) > s.Retencion {
			delete(s.sitios, id)
			n++
		}
	}
	return n
}
