// Package releasediff compara el tarball de una versión con su tag de git y marca lo que
// la puerta trasera de xz (2024) aprovechó: archivos que solo existen en el tarball, scripts
// de build que cambian entre git y tarball, y binarios opacos dentro de los tests.
package releasediff

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"unicode/utf8"
)

// Tree es el contenido de una versión: ruta → bytes.
type Tree map[string][]byte

type Kind int

const (
	OnlyInTarball Kind = iota // existe en el tarball pero no en git
	BuildChanged              // script de build distinto entre git y tarball
	OpaqueBinary              // binario en tests sin un script que lo genere
)

func (k Kind) String() string {
	return [...]string{"solo en el tarball", "script de build cambiado", "binario opaco en tests"}[k]
}

type Finding struct {
	Kind Kind
	Path string // en binarios se agrupan por carpeta y extensión: tests/files/*.xz
	N    int    // cuántos archivos agrupa
}

// generados que autotools mete en el tarball y que no viven en git: se avisan aparte, no bloquean.
var generated = map[string]bool{
	"configure": true, "Makefile.in": true, "aclocal.m4": true, "config.h.in": true,
	"ltmain.sh": true, "compile": true, "missing": true, "install-sh": true, "depcomp": true,
	"config.guess": true, "config.sub": true, "test-driver": true,
}

func isBuild(p string) bool {
	b := path.Base(p)
	switch {
	case b == "configure" || b == "configure.ac" || b == "CMakeLists.txt" || b == "Makefile.am" || b == "meson.build":
		return true
	case strings.HasSuffix(b, ".m4") || strings.HasSuffix(b, ".cmake") || strings.HasSuffix(b, ".sh"):
		return true
	}
	return false
}

func isTest(p string) bool {
	for _, d := range strings.Split(p, "/") {
		if d == "tests" || d == "test" || d == "testdata" {
			return true
		}
	}
	return false
}

func isBinary(b []byte) bool {
	return bytes.IndexByte(b, 0) >= 0 || !utf8.Valid(b)
}

// Compare devuelve los hallazgos que bloquean la versión y, aparte, los generados de autotools.
func Compare(git, tarball Tree) (findings []Finding, generatedOnly []string) {
	var paths []string
	for p := range tarball {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	// texto de todo el árbol, para buscar si algún script menciona (genera) un binario
	var scripts strings.Builder
	for p, b := range tarball {
		if !isBinary(b) && (isBuild(p) || strings.HasSuffix(p, ".py") || strings.HasSuffix(p, ".c") || strings.Contains(path.Base(p), "gen")) {
			scripts.Write(b)
		}
	}

	groups := map[string]int{}
	for _, p := range paths {
		tb := tarball[p]
		gb, inGit := git[p]
		switch {
		case !inGit && generated[path.Base(p)]:
			generatedOnly = append(generatedOnly, p)
		case !inGit:
			findings = append(findings, Finding{Kind: OnlyInTarball, Path: p, N: 1})
		case isBuild(p) && !bytes.Equal(gb, tb):
			findings = append(findings, Finding{Kind: BuildChanged, Path: p, N: 1})
		}
		if isTest(p) && isBinary(tb) && !strings.Contains(scripts.String(), path.Base(p)) {
			groups[path.Dir(p)+"/*"+path.Ext(p)]++
		}
	}
	var gs []string
	for g := range groups {
		gs = append(gs, g)
	}
	sort.Strings(gs)
	for _, g := range gs {
		findings = append(findings, Finding{Kind: OpaqueBinary, Path: g, N: groups[g]})
	}
	sort.SliceStable(findings, func(i, j int) bool { return findings[i].Kind < findings[j].Kind })
	return findings, generatedOnly
}

// ReadTarGz lee un .tar.gz y quita el primer directorio (proyecto-1.2/…).
func ReadTarGz(r io.Reader) (Tree, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return readTar(zr, true)
}

func readTar(r io.Reader, stripTop bool) (Tree, error) {
	t := Tree{}
	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return t, nil
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		name := h.Name
		if stripTop {
			if i := strings.IndexByte(name, '/'); i >= 0 {
				name = name[i+1:]
			}
		}
		b, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		t[name] = b
	}
}

// ReadGitTag lee el contenido de un tag con `git archive` (lo que se revisa en git).
func ReadGitTag(repo, tag string) (Tree, error) {
	cmd := exec.Command("git", "-C", repo, "archive", "--format=tar", tag)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git archive %s: %w", tag, err)
	}
	return readTar(&out, false)
}
