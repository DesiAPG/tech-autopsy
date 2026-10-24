// Mini almacén direccionado por contenido, inspirado en lo que Anthropic publicó sobre Scion.
// No es Scion ni imita su API: es el concepto (changesets, objetos inmutables por hash,
// caché en niveles y checkout de "pide lo que quieres") en unas 300 líneas de Go.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Usage = func() { fmt.Fprintln(os.Stderr, "uso: go run . checkout main [ruta ...]") }
	flag.Parse()
	if flag.NArg() < 2 || flag.Arg(0) != "checkout" {
		flag.Usage()
		os.Exit(2)
	}
	branch, only := flag.Arg(1), flag.Args()[2:]

	srv := NewServer()
	regional := NewCache("scion-cache (regional)", 4096, srv)
	daemon := NewCache("scion-daemon (máquina)", 4096, regional)

	// Historial: un repo de 400 archivos, luego un changeset que toca 3.
	srv.Commit("main", seedRepo(400))
	agentA := NewClient(daemon)
	if _, err := agentA.Checkout(srv.Ref("main")); err != nil {
		fail(err)
	}
	agentA.Requested, agentA.BytesDownloads = 0, 0

	srv.Commit("main", map[string][]byte{
		"src/api/server.go":  []byte("package api // v2: nuevo handler\n"),
		"src/store/cache.go": []byte("package store // v2: LRU\n"),
		"docs/README.md":     []byte("# docs v2\n"),
	})
	if branch != "main" {
		fail(fmt.Errorf("solo existe la rama main en esta demo"))
	}

	fmt.Printf("$ go run . checkout %s\n\n", branch)

	// El agente A ya tenía la versión anterior: solo baja lo que cambió.
	n, err := agentA.Checkout(srv.Ref(branch), only...)
	if err != nil {
		fail(err)
	}
	totalObj, totalBytes := srv.TotalBytes()
	fmt.Printf("agente A (ya tenía la versión anterior) · %d archivos en el checkout\n", n)
	fmt.Printf("  objetos pedidos: solo los que faltan · %d de %d\n", agentA.Requested, totalObj)
	fmt.Printf("  bytes bajados ≪ clon completo · %d B vs %d B\n\n", agentA.BytesDownloads, totalBytes)

	// El agente B arranca vacío, en la misma máquina: casi todo sale del daemon.
	before := daemon.Hits
	agentB := NewClient(daemon)
	n, err = agentB.Checkout(srv.Ref(branch), only...)
	if err != nil {
		fail(err)
	}
	fmt.Printf("agente B (vacío, misma máquina) · %d archivos\n", n)
	fmt.Printf("  aciertos de caché ✓ (inmutables) · %d de %d desde %s\n\n", daemon.Hits-before, agentB.Requested, daemon.Name())

	fmt.Printf("lecturas al servidor en total: %d · el resto lo respondieron las cachés\n", srv.Reads)
}

func seedRepo(n int) map[string][]byte {
	dirs := []string{"src/api", "src/store", "src/web", "docs", "infra", "tests"}
	files := map[string][]byte{}
	for i := 0; i < n; i++ {
		d := dirs[i%len(dirs)]
		files[fmt.Sprintf("%s/file%03d.go", d, i)] = []byte(fmt.Sprintf("package x // archivo %d\n%s", i, pad(i)))
	}
	files["src/api/server.go"] = []byte("package api // v1\n")
	files["src/store/cache.go"] = []byte("package store // v1\n")
	files["docs/README.md"] = []byte("# docs v1\n")
	return files
}

func pad(i int) string {
	b := make([]byte, 600+i%400)
	for k := range b {
		b[k] = byte('a' + (i+k)%26)
	}
	return string(b)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
