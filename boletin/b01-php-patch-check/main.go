// php-patch-check: ¿tu PHP ya tiene el parche de PHP2Shell (CVE-2026-109618 y CVE-2026-109619)?
// Solo revisa versiones y configuración; no contiene ni prueba ningún exploit.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	version := flag.String("version", "", "versión a revisar (por defecto, la del `php` del PATH)")
	php := flag.String("php", "php", "binario de PHP a consultar")
	flag.Parse()

	ini := Ini{}
	v := *version
	if v == "" {
		out, err := exec.Command(*php, "-r", `echo PHP_VERSION, "\n", var_export(ini_get("fatal_error_backtraces"), true), "\n", var_export(ini_get("session.upload_progress.enabled"), true);`).Output()
		if err != nil {
			fmt.Fprintln(os.Stderr, "no pude ejecutar php; usa --version X.Y.Z")
			os.Exit(2)
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		v = lines[0]
		if len(lines) == 3 {
			// ini_get devuelve false si la directiva no existe en esa versión
			if lines[1] != "false" {
				ini["fatal_error_backtraces"] = strings.Trim(lines[1], "'")
			}
			if lines[2] != "false" {
				ini["session.upload_progress.enabled"] = strings.Trim(lines[2], "'")
			}
		}
	}

	r, err := Check(v)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	code := 0
	switch r.Status {
	case Patched:
		fmt.Printf("%s  ✓ parche aplicado\n", r.Version)
	case Vulnerable:
		fmt.Printf("%s  ✕ sin parche · actualiza a %s o superior\n", r.Version, r.Fixed)
		code = 1
	case Unsupported:
		fmt.Printf("%s  ✕ rama sin soporte oficial\n", r.Version)
		code = 1
	}
	if r.Note != "" {
		fmt.Println("   nota:", r.Note)
	}
	if r.Status != Patched {
		fmt.Println("   ojo: algunas distribuciones parchan sin cambiar la versión; revisa el aviso de tu proveedor")
	}
	for _, m := range Mitigations(ini) {
		fmt.Println("   mitigación:", m)
	}
	os.Exit(code)
}
