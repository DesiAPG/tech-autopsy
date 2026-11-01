package releasediff

import "testing"

func TestCompare(t *testing.T) {
	git, tarball := Demo()
	fs, gen := Compare(git, tarball)
	want := []Finding{
		{OnlyInTarball, "m4/build-to-host.m4", 1},
		{BuildChanged, "configure", 1},
		{OpaqueBinary, "tests/files/*.xz", 2},
	}
	if len(fs) != len(want) {
		t.Fatalf("hallazgos = %v, quiero %v", fs, want)
	}
	for i := range want {
		if fs[i] != want[i] {
			t.Errorf("hallazgo %d = %v, quiero %v", i, fs[i], want[i])
		}
	}
	if len(gen) != 0 {
		t.Errorf("generados = %v, quiero ninguno", gen)
	}
}

func TestAutotoolsGeneratedDoNotBlock(t *testing.T) {
	git := Tree{"configure.ac": []byte("AC_INIT\n")}
	tar := Tree{"configure.ac": git["configure.ac"], "Makefile.in": []byte("# automake\n")}
	fs, gen := Compare(git, tar)
	if len(fs) != 0 || len(gen) != 1 || gen[0] != "Makefile.in" {
		t.Fatalf("hallazgos = %v, generados = %v", fs, gen)
	}
}

// En xz, los binarios de prueba también estaban en git: son opacos aunque el tarball coincida.
func TestIdenticalReleaseStillFlagsBinaries(t *testing.T) {
	git, _ := Demo()
	fs, _ := Compare(git, git)
	if len(fs) != 1 || fs[0].Kind != OpaqueBinary {
		t.Fatalf("una versión idéntica a git solo debería marcar los binarios de prueba: %v", fs)
	}
}

func TestBinaryWithGenerator(t *testing.T) {
	git := Tree{"tests/gen-fixtures.sh": []byte("printf '\\0' > tests/files/ok.bin\n")}
	tar := Tree{"tests/gen-fixtures.sh": git["tests/gen-fixtures.sh"], "tests/files/ok.bin": {0, 1, 2}}
	git["tests/files/ok.bin"] = tar["tests/files/ok.bin"]
	if fs, _ := Compare(git, tar); len(fs) != 0 {
		t.Fatalf("un binario que un script regenera no es opaco: %v", fs)
	}
}
