package main

import "testing"

func TestCheck(t *testing.T) {
	cases := []struct {
		in   string
		want Status
	}{
		{"8.5.12", Patched},
		{"8.5.11", Vulnerable},
		{"8.4.27", Patched},
		{"8.4.26", Vulnerable},
		{"8.3.36", Patched},
		{"8.3.40", Patched},
		{"8.2.35", Patched},
		{"8.2.34", Vulnerable},
		{"8.1.31", Unsupported},
		{"7.4.33", Unsupported},
		{"5.6.40", Unsupported},
		{"8.6.0", Patched},
		{"PHP 8.4.10 (cli)", Vulnerable},
		{"8.3.6-0ubuntu0.24.04.5", Vulnerable},
	}
	for _, c := range cases {
		r, err := Check(c.in)
		if err != nil {
			t.Fatalf("%s: %v", c.in, err)
		}
		if r.Status != c.want {
			t.Errorf("%s: estado %d, quería %d", c.in, r.Status, c.want)
		}
	}
	if _, err := Check("ocho"); err == nil {
		t.Error("debía fallar con una versión inválida")
	}
}

func TestBackportNote(t *testing.T) {
	r, _ := Check("8.1.2")
	if r.Note == "" {
		t.Error("8.1 debe mencionar el backport de código fuente")
	}
	r, _ = Check("7.3.0")
	if r.Note != "" {
		t.Error("7.3 no tiene backport publicado")
	}
}

func TestMitigations(t *testing.T) {
	m := Mitigations(Ini{"fatal_error_backtraces": "0", "session.upload_progress.enabled": "1"})
	if len(m) != 2 || m[0] != "fatal_error_backtraces=0 ✓" {
		t.Errorf("mitigaciones: %v", m)
	}
}
