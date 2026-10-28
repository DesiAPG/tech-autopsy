package multiplayer

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Client aplica sus cambios al instante y los manda al servidor. Mientras un cambio suyo no está
// confirmado, ignora lo que llegue del servidor para esa misma propiedad (así la pantalla no parpadea).
type Client struct {
	ID   int
	Name string
	url  string

	mu      sync.Mutex
	doc     Doc
	seq     int
	nextObj int
	pending map[string]Msg // propiedad → último cambio propio sin confirmar
	ws      *WSConn
	done    chan struct{}
}

func NewClient(id int, name, url string) *Client {
	return &Client{ID: id, Name: name, url: url, doc: Doc{}, pending: map[string]Msg{}}
}

// NewObjectID genera IDs que nunca chocan, ni siquiera offline: llevan el ID del cliente como prefijo.
func (c *Client) NewObjectID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nextObj++
	return fmt.Sprintf("%d:%d", c.ID, c.nextObj)
}

// Set edita una propiedad. Offline, el cambio queda pendiente hasta reconectar.
func (c *Client) Set(obj, prop, val string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seq++
	m := Msg{Type: "change", From: c.ID, Seq: c.seq, Obj: obj, Prop: prop, Val: val}
	c.doc.Set(obj, prop, val)
	c.pending[key(obj, prop)] = m
	if c.ws == nil {
		return nil
	}
	return c.ws.WriteMsg(m)
}

// Connect (y reconectar): baja el documento fresco del servidor y reaplica encima los cambios
// propios que siguen pendientes, como hace Figma al volver de offline.
func (c *Client) Connect() error {
	ws, err := Dial(c.url)
	if err != nil {
		return err
	}
	if err := ws.WriteMsg(Msg{Type: "hello", From: c.ID}); err != nil {
		return err
	}
	snap, err := ws.ReadMsg()
	if err != nil || snap.Type != "snapshot" {
		ws.Close()
		return fmt.Errorf("sin snapshot: %v", err)
	}
	c.mu.Lock()
	c.doc = snap.Doc
	if c.doc == nil {
		c.doc = Doc{}
	}
	keys := make([]string, 0, len(c.pending))
	for k := range c.pending {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return c.pending[keys[i]].Seq < c.pending[keys[j]].Seq })
	for _, k := range keys {
		m := c.pending[k]
		c.seq++
		m.Seq = c.seq
		c.pending[k] = m
		c.doc.Set(m.Obj, m.Prop, m.Val)
		if err := ws.WriteMsg(m); err != nil {
			c.mu.Unlock()
			return err
		}
	}
	c.ws, c.done = ws, make(chan struct{})
	c.mu.Unlock()
	go c.readLoop(ws, c.done)
	return nil
}

// Disconnect simula quedarse sin internet: el cliente sigue editando en local.
func (c *Client) Disconnect() {
	c.mu.Lock()
	ws, done := c.ws, c.done
	c.ws = nil
	c.mu.Unlock()
	if ws != nil {
		ws.Close()
		<-done
	}
}

func (c *Client) readLoop(ws *WSConn, done chan struct{}) {
	defer close(done)
	for {
		m, err := ws.ReadMsg()
		if err != nil {
			return
		}
		if m.Type == "change" {
			c.receive(m)
		}
	}
}

func (c *Client) receive(m Msg) {
	c.mu.Lock()
	defer c.mu.Unlock()
	k := key(m.Obj, m.Prop)
	p, mine := c.pending[k]
	switch {
	case m.From == c.ID: // confirmación de un cambio propio
		if mine && p.Seq == m.Seq {
			delete(c.pending, k)
			c.doc.Set(m.Obj, m.Prop, m.Val)
		}
	case mine: // choca con un cambio propio sin confirmar: se ignora
	default:
		c.doc.Set(m.Obj, m.Prop, m.Val)
	}
}

// Doc devuelve una copia del documento local.
func (c *Client) Doc() Doc { c.mu.Lock(); defer c.mu.Unlock(); return c.doc.Clone() }

// Pending cuenta los cambios propios sin confirmar.
func (c *Client) Pending() int { c.mu.Lock(); defer c.mu.Unlock(); return len(c.pending) }

// Settle espera hasta que los clientes no tengan cambios sin confirmar y vean lo mismo que el servidor.
func Settle(s *Server, timeout time.Duration, cs ...*Client) bool {
	end := time.Now().Add(timeout)
	for time.Now().Before(end) {
		ok, sd := true, s.Doc()
		for _, c := range cs {
			if c.Pending() > 0 || !c.Doc().Equal(sd) {
				ok = false
			}
		}
		if ok {
			return true
		}
		time.Sleep(2 * time.Millisecond)
	}
	return false
}
