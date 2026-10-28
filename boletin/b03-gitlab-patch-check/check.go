package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Parches de CVE-2026-110264 (regex, CVSS 10.0), GitLab Critical Patch Release del 2026-10-21.
// Incluyen también el arreglo de CVE-2026-97229 (regex de CI/CD, CVSS 10.0) del 2026-10-12.
var patched = map[string]int{
	"19.5":  1,
	"19.4":  3,
	"19.3":  6,
	"19.2":  9,
	"19.1":  9,
	"19.0":  10,
	"18.11": 13,
}

type Status int

const (
	Patched     Status = iota // línea con parche y versión >= parche
	Vulnerable                // línea con parche pero versión anterior
	NoLine                    // 7.0–18.10: afectada y sin parche en su línea, hay que subir de línea
	NotAffected               // anterior a 7.0
)

type Result struct {
	Version string
	Line    string
	Status  Status
	Fixed   string
}

// Check clasifica una versión de GitLab ("19.4.1", "19.4.1-ee", "v18.11.12").
func Check(raw string) (Result, error) {
	v := strings.TrimPrefix(strings.TrimSpace(raw), "v")
	if i := strings.IndexAny(v, " -+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) < 3 {
		return Result{}, fmt.Errorf("versión no reconocida: %q (se espera X.Y.Z)", raw)
	}
	n := make([]int, 3)
	for i := range n {
		x, err := strconv.Atoi(parts[i])
		if err != nil {
			return Result{}, fmt.Errorf("versión no reconocida: %q", raw)
		}
		n[i] = x
	}
	line := fmt.Sprintf("%d.%d", n[0], n[1])
	r := Result{Version: fmt.Sprintf("%d.%d.%d", n[0], n[1], n[2]), Line: line}
	switch fix, ok := patched[line]; {
	case ok:
		r.Fixed = fmt.Sprintf("%s.%d", line, fix)
		if n[2] >= fix {
			r.Status = Patched
		} else {
			r.Status = Vulnerable
		}
	case n[0] > 19 || (n[0] == 19 && n[1] > 5):
		r.Status = Patched // línea posterior al aviso: ya trae el arreglo
	case n[0] < 7:
		r.Status = NotAffected
	default:
		r.Status = NoLine
		r.Fixed = "18.11.13"
	}
	return r, nil
}
