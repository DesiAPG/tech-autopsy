// gitlab-patch-check: ¿tu GitLab autoalojado ya tiene el arreglo de CVE-2026-110264 (y de CVE-2026-97229)?
// Solo compara versiones; no contiene ni prueba ningún exploit.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// version lee /api/v4/version (pide un token con alcance read_api o read_user).
func version(base, token string) (string, error) {
	req, err := http.NewRequest("GET", strings.TrimRight(base, "/")+"/api/v4/version", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	res, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", fmt.Errorf("%s respondió %s", base, res.Status)
	}
	var v struct{ Version string }
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		return "", err
	}
	return v.Version, nil
}

func main() {
	ver := flag.String("version", "", "versión a revisar, por ejemplo 19.4.1-ee")
	url := flag.String("url", "", "URL de tu GitLab (lee la versión de /api/v4/version con $GITLAB_TOKEN)")
	flag.Parse()

	v := *ver
	if v == "" && *url != "" {
		var err error
		if v, err = version(*url, os.Getenv("GITLAB_TOKEN")); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	if v == "" {
		fmt.Fprintln(os.Stderr, "usa --version X.Y.Z o --url https://tu-gitlab (con GITLAB_TOKEN)")
		os.Exit(2)
	}
	r, err := Check(v)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	code := 0
	switch r.Status {
	case Patched:
		fmt.Printf("%s  ✓ tiene el arreglo de CVE-2026-110264\n", r.Version)
	case Vulnerable:
		fmt.Printf("%s  ✕ sin arreglo · actualiza a %s o superior\n", r.Version, r.Fixed)
		code = 1
	case NoLine:
		fmt.Printf("%s  ✕ afectada y su línea no tiene parche · sube al menos a %s\n", r.Version, r.Fixed)
		code = 1
	case NotAffected:
		fmt.Printf("%s  · anterior a 7.0, fuera del rango afectado (y sin soporte)\n", r.Version)
	}
	if r.Status == Vulnerable && (r.Line == "19.4" || r.Line == "19.3") {
		fmt.Println("   ojo: este parche trae migraciones; en un solo nodo GitLab no arranca hasta que terminan")
	}
	fmt.Println("   GitLab.com ya está parchado; esto es para instancias autoalojadas (CE/EE)")
	os.Exit(code)
}
