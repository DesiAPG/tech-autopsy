package multiplayer

import (
	"net/http"
	"sync"
)

// Server es la autoridad: aplica los cambios en el orden en que llegan y los reenvía a todos,
// incluido quien lo mandó (le sirve de confirmación). Por eso no hacen falta relojes ni consenso.
type Server struct {
	mu    sync.Mutex
	doc   Doc
	conns map[*WSConn]bool
	log   []Msg
}

func NewServer() *Server { return &Server{doc: Doc{}, conns: map[*WSConn]bool{}} }

// ServeHTTP acepta un cliente: hello → snapshot del documento → cambios en los dos sentidos.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws, err := Upgrade(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer ws.Close()
	if m, err := ws.ReadMsg(); err != nil || m.Type != "hello" {
		return
	}
	// snapshot y alta bajo el mismo candado: ningún cambio se cuela entre los dos
	s.mu.Lock()
	err = ws.WriteMsg(Msg{Type: "snapshot", Doc: s.doc.Clone()})
	s.conns[ws] = true
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.conns, ws); s.mu.Unlock() }()
	for err == nil {
		var m Msg
		if m, err = ws.ReadMsg(); err == nil && m.Type == "change" {
			s.apply(m)
		}
	}
}

// apply: last-writer-wins por propiedad. El último valor que llega a esta línea es el que queda.
func (s *Server) apply(m Msg) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.doc.Set(m.Obj, m.Prop, m.Val)
	s.log = append(s.log, m)
	for c := range s.conns {
		_ = c.WriteMsg(m) // bajo el candado: todos reciben los cambios en el mismo orden
	}
}

// Doc devuelve una copia del documento del servidor.
func (s *Server) Doc() Doc { s.mu.Lock(); defer s.mu.Unlock(); return s.doc.Clone() }

// Log devuelve, en orden de llegada, los cambios que recibió el servidor para una propiedad.
func (s *Server) Log(obj, prop string) []Msg {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Msg
	for _, m := range s.log {
		if m.Obj == obj && m.Prop == prop {
			out = append(out, m)
		}
	}
	return out
}
