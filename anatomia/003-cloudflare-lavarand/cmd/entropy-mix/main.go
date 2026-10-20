// entropy-mix mezcla varias fuentes de entropía como LavaRand y revisa tu
// código en busca de math/rand usado para secretos.
//
//	entropy-mix demo                 # 3 fuentes, el atacante conoce 2: no puede predecir la salida
//	entropy-mix lint archivo.go ...  # sale con 1 si encuentra math/rand en tokens, claves…
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/bits"
	"os"

	entropymix "github.com/DesiAPG/tech-autopsy/anatomia/003-cloudflare-lavarand"
)

const info = "tech-autopsy/003 lavarand demo"

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "demo":
		os.Exit(demo(os.Stdout))
	case "lint":
		os.Exit(lint(os.Args[2:], os.Stdout))
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "uso: entropy-mix demo | entropy-mix lint archivo.go...")
	os.Exit(2)
}

func demo(w io.Writer) int {
	camera := entropymix.Frame(1.7)
	evil := sha256.Sum256([]byte("fuente controlada por el atacante"))
	local := entropymix.Local(32)

	real, err := entropymix.Mix(info, 32, camera, evil[:], local)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	// El atacante robó la foto y controla una fuente, pero no conoce la local.
	guess, _ := entropymix.Mix(info, 32, camera, evil[:], make([]byte, 32))
	// Si las compromete todas, sí puede reproducir la salida.
	all, _ := entropymix.Mix(info, 32, camera, evil[:], local)

	fmt.Fprintln(w, "fuente        bytes   ¿el atacante la tiene?")
	fmt.Fprintf(w, "cámara        %5d   sí · robó la foto\n", len(camera))
	fmt.Fprintf(w, "comprometida  %5d   sí · la controla\n", len(evil))
	fmt.Fprintf(w, "local         %5d   no\n", len(local))
	fmt.Fprintf(w, "mezcla HKDF-SHA256 → %s\n", short(real))
	fmt.Fprintf(w, "atacante (2 de 3)  → %s  %s\n", short(guess), verdict(guess, real))
	fmt.Fprintf(w, "atacante (3 de 3)  → %s  %s\n", short(all), verdict(all, real))
	fmt.Fprintf(w, "bits distintos: %d/256\n", diff(guess, real))
	fmt.Fprintln(w, "basta UNA fuente secreta ✓")
	return 0
}

func short(b []byte) string {
	h := hex.EncodeToString(b)
	return h[:4] + "…" + h[len(h)-4:]
}

func verdict(a, b []byte) string {
	if bytes.Equal(a, b) {
		return "✓ igual"
	}
	return "✖ falla"
}

func diff(a, b []byte) int {
	n := 0
	for i := range a {
		n += bits.OnesCount8(a[i] ^ b[i])
	}
	return n
}

func lint(paths []string, w io.Writer) int {
	if len(paths) == 0 {
		usage()
	}
	found := 0
	for _, p := range paths {
		fs, err := entropymix.Lint(p, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		for _, f := range fs {
			fmt.Fprintln(w, f)
		}
		found += len(fs)
	}
	if found > 0 {
		fmt.Fprintf(w, "%d hallazgo(s). Usa crypto/rand\n", found)
		return 1
	}
	return 0
}
