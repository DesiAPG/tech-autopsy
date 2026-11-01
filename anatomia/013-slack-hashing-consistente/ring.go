// Package ring es un anillo de hashing consistente con nodos virtuales, como el que usa Slack para
// decidir qué Channel Server es dueño de cada canal: el ID cae en un punto del anillo y el siguiente
// servidor en el sentido del reloj es su dueño. Si un servidor sale, solo se mueven sus canales.
package ring

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
)

// Ring reparte claves entre servidores sin una tabla central.
type Ring struct {
	vnodes int               // puntos por servidor: más puntos, reparto más parejo
	points []uint64          // posiciones en el anillo, ordenadas
	owner  map[uint64]string // posición → servidor
}

// New crea un anillo con vnodes puntos por servidor.
func New(vnodes int, servers ...string) *Ring {
	r := &Ring{vnodes: vnodes, owner: map[uint64]string{}}
	for _, s := range servers {
		r.Add(s)
	}
	return r
}

func hash(s string) uint64 {
	h := sha256.Sum256([]byte(s))
	return binary.BigEndian.Uint64(h[:8])
}

// Add mete un servidor en el anillo (sus vnodes puntos).
func (r *Ring) Add(server string) {
	for i := 0; i < r.vnodes; i++ {
		p := hash(fmt.Sprintf("%s#%d", server, i))
		r.owner[p] = server
		r.points = append(r.points, p)
	}
	sort.Slice(r.points, func(i, j int) bool { return r.points[i] < r.points[j] })
}

// Remove saca un servidor: sus tramos pasan al siguiente servidor de cada uno.
func (r *Ring) Remove(server string) {
	kept := r.points[:0]
	for _, p := range r.points {
		if r.owner[p] == server {
			delete(r.owner, p)
			continue
		}
		kept = append(kept, p)
	}
	r.points = kept
}

// Owner devuelve el servidor dueño de la clave: el primer punto del anillo en o después de su hash.
func (r *Ring) Owner(key string) string {
	if len(r.points) == 0 {
		return ""
	}
	h := hash(key)
	i := sort.Search(len(r.points), func(i int) bool { return r.points[i] >= h })
	if i == len(r.points) {
		i = 0 // da la vuelta al anillo
	}
	return r.owner[r.points[i]]
}

// ModN es la alternativa ingenua: hash(clave) % N. Al cambiar N se mueve casi todo.
func ModN(key string, servers []string) string {
	return servers[hash(key)%uint64(len(servers))]
}
