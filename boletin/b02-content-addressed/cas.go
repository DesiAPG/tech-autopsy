package main

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// ID es el hash del contenido: si el contenido no cambia, el ID tampoco, y viceversa.
type ID string

func hashOf(b []byte) ID {
	s := sha256.Sum256(b)
	return ID(hex.EncodeToString(s[:]))
}

// Fetcher es cualquier cosa que sabe devolver un objeto por su ID: servidor, caché o daemon.
// Todos responden la misma "RPC", así que el cliente no necesita conocer la topología.
type Fetcher interface {
	Get(id ID) ([]byte, bool)
	Name() string
}

// Server es la fuente de verdad. Commits y merges pasan aquí: el cliente no hace push.
type Server struct {
	objects map[ID][]byte
	refs    map[string]ID // rama → árbol raíz
	Reads   int
}

func NewServer() *Server { return &Server{objects: map[ID][]byte{}, refs: map[string]ID{}} }

func (s *Server) Name() string { return "server" }

func (s *Server) Get(id ID) ([]byte, bool) {
	s.Reads++
	b, ok := s.objects[id]
	return b, ok
}

func (s *Server) put(b []byte) ID {
	id := hashOf(b)
	s.objects[id] = b
	return id
}

// Changeset: solo los archivos que cambiaron respecto al padre, con el hash de su contenido.
type Changeset struct {
	Parent  ID
	Changed map[string]ID // ruta → blob
}

// Commit aplica un changeset en el servidor y mueve la rama. Devuelve el nuevo árbol raíz.
func (s *Server) Commit(branch string, files map[string][]byte) (Changeset, ID) {
	cs := Changeset{Parent: s.refs[branch], Changed: map[string]ID{}}
	current := map[string]ID{}
	if cs.Parent != "" {
		s.flatten(cs.Parent, "", current)
	}
	for path, content := range files {
		id := s.put(content)
		cs.Changed[path] = id
		current[path] = id
	}
	root := s.buildTree(current)
	s.refs[branch] = root
	return cs, root
}

func (s *Server) Ref(branch string) ID { return s.refs[branch] }

// TotalBytes es lo que bajaría un clon completo: todos los objetos de todo el historial.
func (s *Server) TotalBytes() (objects, bytes int) {
	for _, b := range s.objects {
		objects++
		bytes += len(b)
	}
	return
}

// Un árbol se serializa como líneas "tipo id nombre", ordenadas: mismo contenido, mismo hash.
func (s *Server) buildTree(files map[string]ID) ID {
	type entry struct {
		kind, name string
		id         ID
	}
	dirs := map[string]map[string]ID{}
	var entries []entry
	for path, id := range files {
		if i := strings.IndexByte(path, '/'); i >= 0 {
			d := path[:i]
			if dirs[d] == nil {
				dirs[d] = map[string]ID{}
			}
			dirs[d][path[i+1:]] = id
		} else {
			entries = append(entries, entry{"blob", path, id})
		}
	}
	for d, sub := range dirs {
		entries = append(entries, entry{"tree", d, s.buildTree(sub)})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	var b strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&b, "%s %s %s\n", e.kind, e.id, e.name)
	}
	return s.put([]byte(b.String()))
}

func (s *Server) flatten(tree ID, prefix string, out map[string]ID) {
	for _, e := range parseTree(s.objects[tree]) {
		if e.kind == "tree" {
			s.flatten(e.id, prefix+e.name+"/", out)
		} else {
			out[prefix+e.name] = e.id
		}
	}
}

type treeEntry struct {
	kind, name string
	id         ID
}

func parseTree(b []byte) []treeEntry {
	var out []treeEntry
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if line == "" {
			continue
		}
		p := strings.SplitN(line, " ", 3)
		out = append(out, treeEntry{p[0], p[2], ID(p[1])})
	}
	return out
}

// Cache LRU delante de otro Fetcher. Como los objetos son inmutables, nunca hay que invalidar.
type Cache struct {
	name         string
	next         Fetcher
	cap          int
	ll           *list.List
	items        map[ID]*list.Element
	Hits, Misses int
}

type cacheItem struct {
	id ID
	b  []byte
}

func NewCache(name string, capacity int, next Fetcher) *Cache {
	return &Cache{name: name, next: next, cap: capacity, ll: list.New(), items: map[ID]*list.Element{}}
}

func (c *Cache) Name() string { return c.name }

func (c *Cache) Get(id ID) ([]byte, bool) {
	if el, ok := c.items[id]; ok {
		c.Hits++
		c.ll.MoveToFront(el)
		return el.Value.(cacheItem).b, true
	}
	c.Misses++
	b, ok := c.next.Get(id)
	if !ok {
		return nil, false
	}
	c.items[id] = c.ll.PushFront(cacheItem{id, b})
	if c.ll.Len() > c.cap {
		last := c.ll.Back()
		c.ll.Remove(last)
		delete(c.items, last.Value.(cacheItem).id)
	}
	return b, true
}

// Client guarda objetos locales y hace checkout pidiendo solo lo que le falta.
// No le dice al servidor lo que tiene: pide por hash lo que quiere.
type Client struct {
	local          map[ID][]byte
	via            Fetcher
	Requested      int
	BytesDownloads int
}

func NewClient(via Fetcher) *Client { return &Client{local: map[ID][]byte{}, via: via} }

func (c *Client) get(id ID) ([]byte, error) {
	if b, ok := c.local[id]; ok {
		return b, nil
	}
	b, ok := c.via.Get(id)
	if !ok {
		return nil, fmt.Errorf("objeto %s no existe", id[:8])
	}
	if hashOf(b) != id {
		return nil, fmt.Errorf("objeto %s corrupto: el hash no coincide", id[:8])
	}
	c.Requested++
	c.BytesDownloads += len(b)
	c.local[id] = b
	return b, nil
}

// Checkout recorre el árbol desde la raíz. Con `only` hace un checkout sparse: solo esas rutas.
func (c *Client) Checkout(root ID, only ...string) (files int, err error) {
	return c.walk(root, "", only)
}

func (c *Client) walk(tree ID, prefix string, only []string) (int, error) {
	b, err := c.get(tree)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range parseTree(b) {
		path := prefix + e.name
		if !wanted(path, e.kind == "tree", only) {
			continue
		}
		if e.kind == "tree" {
			k, err := c.walk(e.id, path+"/", only)
			if err != nil {
				return n, err
			}
			n += k
			continue
		}
		if _, err := c.get(e.id); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func wanted(path string, dir bool, only []string) bool {
	if len(only) == 0 {
		return true
	}
	for _, o := range only {
		o = strings.TrimSuffix(o, "/")
		if path == o || strings.HasPrefix(path, o+"/") || (dir && strings.HasPrefix(o, path+"/")) {
			return true
		}
	}
	return false
}
