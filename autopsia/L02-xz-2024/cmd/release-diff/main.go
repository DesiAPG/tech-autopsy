// release-diff: compara el tarball de una versión con su tag de git antes de compilarlo (xz, 2024).
package main

import (
	"flag"
	"fmt"
	"os"

	releasediff "github.com/DesiAPG/tech-autopsy/autopsia/L02-xz-2024"
)

func main() {
	tarball := flag.String("tarball", "", "tarball de la versión (.tar.gz)")
	tag := flag.String("tag", "", "tag de git que debería corresponderle")
	repo := flag.String("repo", ".", "repositorio git donde está el tag")
	demo := flag.Bool("demo", false, "usa el proyecto de juguete con el patrón de xz 5.6.0")
	flag.Parse()

	var git, tar releasediff.Tree
	var err error
	switch {
	case *demo:
		git, tar = releasediff.Demo()
		fmt.Println("$ release-diff --tarball proyecto-1.2.tar.gz --tag v1.2")
	case *tarball != "" && *tag != "":
		f, e := os.Open(*tarball)
		if e != nil {
			fail(e)
		}
		defer f.Close()
		if tar, err = releasediff.ReadTarGz(f); err != nil {
			fail(err)
		}
		if git, err = releasediff.ReadGitTag(*repo, *tag); err != nil {
			fail(err)
		}
	default:
		flag.Usage()
		os.Exit(2)
	}

	fs, gen := releasediff.Compare(git, tar)
	for _, g := range gen {
		fmt.Printf("generado por autotools (revisar): %s\n", g)
	}
	for _, f := range fs {
		fmt.Printf("%-42s✗\n", fmt.Sprintf("%s: %s", f.Kind, f.Path))
	}
	if len(fs) > 0 {
		fmt.Printf("BLOQUEADO: %d diferencias sin explicar\n", len(fs))
		os.Exit(1)
	}
	fmt.Println("OK: el tarball coincide con el tag")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "release-diff:", err)
	os.Exit(2)
}
