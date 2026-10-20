package entropymix

import (
	"bytes"
	"testing"
)

func TestMixDeterministic(t *testing.T) {
	a, _ := Mix("t", 32, []byte("x"), []byte("y"))
	b, _ := Mix("t", 32, []byte("x"), []byte("y"))
	if !bytes.Equal(a, b) {
		t.Fatal("las mismas fuentes deben dar la misma salida")
	}
}

func TestOneSecretSourceChangesOutput(t *testing.T) {
	cam, evil := Frame(0), []byte("conocida")
	a, _ := Mix("t", 32, cam, evil, Local(32))
	b, _ := Mix("t", 32, cam, evil, Local(32))
	if bytes.Equal(a, b) {
		t.Fatal("una fuente secreta distinta debe cambiar la salida")
	}
}

func TestLengthPrefix(t *testing.T) {
	a, _ := Mix("t", 32, []byte("ab"), []byte("c"))
	b, _ := Mix("t", 32, []byte("a"), []byte("bc"))
	if bytes.Equal(a, b) {
		t.Fatal(`("ab","c") y ("a","bc") no deben coincidir`)
	}
}

func TestFrameSize(t *testing.T) {
	if n := len(Frame(0)); n != 30000 {
		t.Fatalf("100×100×3 = 30000 bytes, got %d", n)
	}
}

func TestLint(t *testing.T) {
	fs, err := Lint("testdata/tokens.go.txt", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 2 {
		t.Fatalf("want 2 findings (NewSessionToken, resetCode), got %d: %v", len(fs), fs)
	}
	if fs[0].Why != "NewSessionToken" || fs[1].Why != "otp" {
		t.Fatalf("unexpected findings: %v", fs)
	}
}
