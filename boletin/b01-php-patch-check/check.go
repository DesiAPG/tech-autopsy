package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Parches de PHP2Shell (CVE-2026-109618 y CVE-2026-109619), publicados por php.net el 2026-10-21.
var patched = map[string]int{
	"8.5": 12,
	"8.4": 27,
	"8.3": 36,
	"8.2": 35,
}

// Ramas con backport mínimo de código fuente publicado por el equipo de PHP (sin release oficial).
var sourceBackport = map[string]bool{"7.4": true, "8.0": true, "8.1": true}

type Status int

const (
	Patched     Status = iota // rama con soporte y versión >= parche
	Vulnerable                // rama con soporte pero versión anterior al parche
	Unsupported               // rama sin actualizaciones oficiales (8.1 y anteriores)
)

type Result struct {
	Version string
	Branch  string
	Status  Status
	Fixed   string // versión que trae el parche, si la rama tiene soporte
	Note    string
}

// Check clasifica una versión de PHP ("8.4.27", "8.3.10-1ubuntu2", "PHP 8.2.1 (cli)").
func Check(raw string) (Result, error) {
	v := strings.TrimSpace(raw)
	v = strings.TrimPrefix(v, "PHP ")
	if i := strings.IndexAny(v, " -+~("); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) < 3 {
		return Result{}, fmt.Errorf("versión no reconocida: %q (se espera X.Y.Z)", raw)
	}
	nums := make([]int, 3)
	for i := 0; i < 3; i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return Result{}, fmt.Errorf("versión no reconocida: %q", raw)
		}
		nums[i] = n
	}
	branch := fmt.Sprintf("%d.%d", nums[0], nums[1])
	r := Result{Version: fmt.Sprintf("%d.%d.%d", nums[0], nums[1], nums[2]), Branch: branch}
	if fix, ok := patched[branch]; ok {
		r.Fixed = fmt.Sprintf("%s.%d", branch, fix)
		if nums[2] >= fix {
			r.Status = Patched
		} else {
			r.Status = Vulnerable
		}
		return r, nil
	}
	if nums[0] > 8 || (nums[0] == 8 && nums[1] > 5) {
		// rama más nueva que las anunciadas: se asume que ya incluye el arreglo
		r.Status = Patched
		r.Note = "rama posterior al aviso"
		return r, nil
	}
	r.Status = Unsupported
	if sourceBackport[branch] {
		r.Note = "el equipo de PHP publicó un backport mínimo de código fuente para " + branch
	}
	return r, nil
}

// Mitigaciones temporales del aviso GHSA-6wmf-4rv5-p3rp (no reemplazan al parche).
type Ini map[string]string

func Mitigations(ini Ini) []string {
	var out []string
	off := func(v string) bool {
		v = strings.ToLower(strings.TrimSpace(v))
		return v == "0" || v == "" || v == "off" || v == "false"
	}
	if v, ok := ini["fatal_error_backtraces"]; ok {
		if off(v) {
			out = append(out, "fatal_error_backtraces=0 ✓")
		} else {
			out = append(out, "fatal_error_backtraces activo ✕ (mitigación: =0)")
		}
	}
	if v, ok := ini["session.upload_progress.enabled"]; ok {
		if off(v) {
			out = append(out, "session.upload_progress.enabled=0 ✓")
		} else {
			out = append(out, "session.upload_progress.enabled activo ✕ (mitigación: =0)")
		}
	}
	return out
}
