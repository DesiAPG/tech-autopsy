package main

import "testing"

func TestCheck(t *testing.T) {
	cases := []struct {
		in   string
		want Status
	}{
		{"19.5.1", Patched},
		{"19.5.0", Vulnerable},
		{"19.4.3-ee", Patched},
		{"19.4.2", Vulnerable}, // tenía el parche del 12-oct, no el del 21
		{"19.3.6", Patched},
		{"19.0.9", Vulnerable},
		{"18.11.13", Patched},
		{"18.11.12", Vulnerable},
		{"18.10.5", NoLine},
		{"7.0.0", NoLine},
		{"6.9.2", NotAffected},
		{"19.6.0", Patched},
		{"v19.1.9", Patched},
	}
	for _, c := range cases {
		r, err := Check(c.in)
		if err != nil {
			t.Fatalf("%s: %v", c.in, err)
		}
		if r.Status != c.want {
			t.Errorf("%s: estado %d, se esperaba %d", c.in, r.Status, c.want)
		}
	}
	if _, err := Check("diecinueve"); err == nil {
		t.Error("debía fallar con una versión inválida")
	}
}
