package main

import "testing"

func TestSameContentSameID(t *testing.T) {
	if hashOf([]byte("hola")) != hashOf([]byte("hola")) || hashOf([]byte("hola")) == hashOf([]byte("chao")) {
		t.Fatal("el ID debe depender solo del contenido")
	}
}

func TestIncrementalCheckoutOnlyFetchesChanges(t *testing.T) {
	srv := NewServer()
	srv.Commit("main", seedRepo(120))
	c := NewClient(srv)
	if _, err := c.Checkout(srv.Ref("main")); err != nil {
		t.Fatal(err)
	}
	c.Requested = 0
	srv.Commit("main", map[string][]byte{"src/api/server.go": []byte("v2")})
	if _, err := c.Checkout(srv.Ref("main")); err != nil {
		t.Fatal(err)
	}
	// 1 blob nuevo + árboles en el camino: raíz, src/, src/api/
	if c.Requested != 4 {
		t.Fatalf("pidió %d objetos, esperaba 4", c.Requested)
	}
}

func TestCacheServesSecondClient(t *testing.T) {
	srv := NewServer()
	srv.Commit("main", seedRepo(60))
	daemon := NewCache("d", 1000, srv)
	a, b := NewClient(daemon), NewClient(daemon)
	if _, err := a.Checkout(srv.Ref("main")); err != nil {
		t.Fatal(err)
	}
	reads := srv.Reads
	if _, err := b.Checkout(srv.Ref("main")); err != nil {
		t.Fatal(err)
	}
	if srv.Reads != reads {
		t.Fatalf("el segundo cliente no debía tocar el servidor (%d lecturas extra)", srv.Reads-reads)
	}
}

func TestSparseCheckout(t *testing.T) {
	srv := NewServer()
	srv.Commit("main", seedRepo(60))
	c := NewClient(srv)
	n, err := c.Checkout(srv.Ref("main"), "docs")
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 || n >= 60 {
		t.Fatalf("checkout sparse de docs/ trajo %d archivos", n)
	}
}

func TestCorruptObjectRejected(t *testing.T) {
	srv := NewServer()
	_, root := srv.Commit("main", map[string][]byte{"a.txt": []byte("a")})
	for id := range srv.objects {
		if id != root {
			srv.objects[id] = []byte("manipulado")
		}
	}
	if _, err := NewClient(srv).Checkout(root); err == nil {
		t.Fatal("un objeto cuyo hash no coincide debe rechazarse")
	}
}
