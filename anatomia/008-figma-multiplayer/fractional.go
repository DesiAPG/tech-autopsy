package multiplayer

// Between devuelve una posición estrictamente entre a y b (fractional indexing). Las posiciones son los
// dígitos decimales de una fracción entre 0 y 1: "3" es 0,3 y "35" es 0,35. "" como a es 0 y "" como b es 1.
// Insertar entre dos capas nunca obliga a renumerar las demás. Requiere a < b.
func Between(a, b string) string {
	digit := func(s string, i int) int {
		if i < len(s) {
			return int(s[i] - '0')
		}
		return 0
	}
	upperOpen := b == "" // el límite de arriba es 1: en esta posición vale 10
	prefix := make([]byte, 0, len(a)+1)
	for i := 0; ; i++ {
		lo, hi := digit(a, i), 10
		if !upperOpen {
			hi = digit(b, i)
		}
		if hi-lo > 1 {
			return string(append(prefix, byte('0'+lo+(hi-lo)/2)))
		}
		prefix = append(prefix, byte('0'+lo))
		if hi-lo == 1 {
			upperOpen = true // ya quedamos por debajo de b: de aquí en adelante el techo es "…999"
		}
	}
}
