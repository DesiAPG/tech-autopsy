// Package multiplayer es un mini servidor multijugador al estilo de Figma: el documento es un mapa
// objeto → propiedad → valor, el servidor central decide el orden y en cada propiedad gana el último
// valor que le llega (last-writer-wins por propiedad).
//
// Fuente: "How Figma's multiplayer technology works", Evan Wallace, 2019-10-16.
// https://www.figma.com/blog/how-figmas-multiplayer-technology-works/
package multiplayer

import (
	"fmt"
	"sort"
	"strings"
)

// Doc es el documento como lo modela Figma: Map<ObjectID, Map<Property, Value>>.
type Doc map[string]map[string]string

// Set escribe una propiedad; crea el objeto si no existe.
func (d Doc) Set(obj, prop, val string) {
	if d[obj] == nil {
		d[obj] = map[string]string{}
	}
	d[obj][prop] = val
}

// Get lee una propiedad ("" si no existe).
func (d Doc) Get(obj, prop string) string { return d[obj][prop] }

// Clone copia el documento entero.
func (d Doc) Clone() Doc {
	out := Doc{}
	for o, props := range d {
		for p, v := range props {
			out.Set(o, p, v)
		}
	}
	return out
}

// Equal compara dos documentos propiedad por propiedad.
func (d Doc) Equal(o Doc) bool { return d.String() == o.String() }

// String lista los objetos en el orden de capas (propiedad "pos", fractional indexing).
func (d Doc) String() string {
	ids := make([]string, 0, len(d))
	for id := range d {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		pi, pj := d[ids[i]]["pos"], d[ids[j]]["pos"]
		if pi != pj {
			return pi < pj
		}
		return ids[i] < ids[j]
	})
	var b strings.Builder
	for _, id := range ids {
		props := make([]string, 0, len(d[id]))
		for p, v := range d[id] {
			props = append(props, p+"="+v)
		}
		sort.Strings(props)
		fmt.Fprintf(&b, "  %-5s %s\n", id, strings.Join(props, "  "))
	}
	return b.String()
}

// Msg es lo que viaja por el WebSocket.
type Msg struct {
	Type string `json:"type"` // hello | snapshot | change
	From int    `json:"from,omitempty"`
	Seq  int    `json:"seq,omitempty"`
	Obj  string `json:"obj,omitempty"`
	Prop string `json:"prop,omitempty"`
	Val  string `json:"val,omitempty"`
	Doc  Doc    `json:"doc,omitempty"`
}

func key(obj, prop string) string { return obj + "\x00" + prop }
