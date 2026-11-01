package releasediff

// Demo arma un proyecto de juguete con el mismo patrón que xz 5.6.0 (sin nada malicioso):
// el tag de git está limpio y el tarball trae un .m4 de más, un script de build distinto
// y dos binarios "de prueba" que ningún script genera.
func Demo() (git, tarball Tree) {
	git = Tree{
		"README.md":              []byte("proyecto de juguete\n"),
		"configure.ac":           []byte("AC_INIT([proyecto], [1.2])\nAC_OUTPUT\n"),
		"configure":              []byte("#!/bin/sh\n# script de configuración versionado en git\n"),
		"src/compress.c":         []byte("int compress(void) { return 0; }\n"),
		"tests/test_basic.sh":    []byte("./compress < tests/files/sample.txt\n"),
		"tests/files/sample.txt": []byte("hola\n"),
		"tests/files/bad-3.xz":   {0xfd, '7', 'z', 'X', 'Z', 0x00, 0x13, 0x37},
		"tests/files/good-2.xz":  {0xfd, '7', 'z', 'X', 'Z', 0x00, 0x42, 0x42},
	}
	tarball = Tree{}
	for p, b := range git {
		tarball[p] = b
	}
	tarball["configure"] = []byte("#!/bin/sh\n# script de configuración versionado en git\n. ./m4/build-to-host.m4\n")
	tarball["m4/build-to-host.m4"] = []byte("dnl macro que solo existe en el tarball\n")
	return git, tarball
}
