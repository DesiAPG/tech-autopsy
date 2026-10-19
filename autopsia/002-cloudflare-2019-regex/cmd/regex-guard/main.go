// regex-guard busca regex con riesgo de backtracking catastrófico y mide cómo
// explota un motor con backtracking frente al motor lineal de Go.
//
//	regex-guard lint reglas.txt          # un patrón por línea; sale con 1 si hay hallazgos
//	regex-guard lint -e '.*(?:.*=.*)'    # un patrón suelto
//	regex-guard bench                    # tabla de pasos y tiempos para .*.*=.*
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	regexguard "github.com/DesiAPG/tech-autopsy/autopsia/002-cloudflare-2019-regex"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "lint":
		os.Exit(lint(os.Args[2:], os.Stdout))
	case "bench":
		os.Exit(bench(os.Args[2:], os.Stdout))
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "uso: regex-guard lint [-e patrón] [archivos...] | regex-guard bench [-pattern p] [-max n]")
	os.Exit(2)
}

type source struct {
	where   string
	pattern string
}

func lint(args []string, w io.Writer) int {
	fs := flag.NewFlagSet("lint", flag.ExitOnError)
	expr := fs.String("e", "", "patrón a revisar")
	fs.Parse(args)

	var srcs []source
	if *expr != "" {
		srcs = append(srcs, source{"-e", *expr})
	}
	for _, path := range fs.Args() {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		sc := bufio.NewScanner(f)
		for n := 1; sc.Scan(); n++ {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			srcs = append(srcs, source{fmt.Sprintf("%s:%d", path, n), line})
		}
		f.Close()
	}
	if len(srcs) == 0 {
		usage()
	}

	bad := 0
	for _, s := range srcs {
		findings, err := regexguard.Lint(s.pattern)
		if err != nil {
			fmt.Fprintf(w, "%s: aviso: %v\n", s.where, err)
			continue
		}
		for _, f := range findings {
			bad++
			fmt.Fprintf(w, "%s: ✖ %s · %s\n    en: %s\n", s.where, f.Rule, f.Message, f.Fragment)
		}
	}
	if bad > 0 {
		fmt.Fprintf(w, "\n%d hallazgo(s). Usa un motor lineal (RE2, regexp de Go, regex de Rust) o reescribe el patrón.\n", bad)
		return 1
	}
	fmt.Fprintf(w, "✓ %d patrón(es) sin riesgo de backtracking detectado\n", len(srcs))
	return 0
}

func bench(args []string, w io.Writer) int {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	pattern := fs.String("pattern", ".*.*=.*", "patrón (solo literales, '.', '*', '^', '$')")
	max := fs.Int("max", 3200, "longitud máxima del texto")
	fs.Parse(args)

	re := regexp.MustCompile(*pattern)
	fmt.Fprintf(w, "patrón %q contra un texto de n 'x' sin '=' (el peor caso)\n\n", *pattern)
	fmt.Fprintf(w, "%8s  %16s  %14s  %14s\n", "n", "pasos backtrack", "backtracking", "regexp de Go")
	for n := 25; n <= *max; n *= 2 {
		text := strings.Repeat("x", n)
		start := time.Now()
		_, steps := regexguard.Backtrack(*pattern, text)
		bt := time.Since(start)
		start = time.Now()
		re.MatchString(text)
		lin := time.Since(start)
		fmt.Fprintf(w, "%8d  %16d  %14s  %14s\n", n, steps, bt.Round(time.Microsecond), lin.Round(time.Microsecond))
	}
	fmt.Fprintln(w, "\nAl duplicar n, el backtracking multiplica su trabajo; el motor lineal solo lo duplica.")
	return 0
}
