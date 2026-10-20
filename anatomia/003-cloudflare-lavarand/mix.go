// Package entropymix reproduce la idea de LavaRand (Cloudflare): mezclar varias
// fuentes de entropía independientes con un hash criptográfico, de modo que la
// salida sea impredecible mientras una sola de ellas siga siendo secreta.
package entropymix

import (
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"math"
)

// Mix combina las fuentes con HKDF-SHA256 y devuelve n bytes. Cada fuente va
// precedida de su longitud, así ("ab", "c") y ("a", "bc") no producen lo mismo.
func Mix(info string, n int, sources ...[]byte) ([]byte, error) {
	var secret []byte
	for _, s := range sources {
		secret = binary.BigEndian.AppendUint64(secret, uint64(len(s)))
		secret = append(secret, s...)
	}
	return hkdf.Key(sha256.New, secret, nil, info, n)
}

// Size de la "foto" simulada: 100×100 píxeles RGB, el ejemplo del post-mortem.
const (
	Width  = 100
	Height = 100
)

// Frame simula una foto de la pared: blobs de lava que se mueven con el tiempo t
// más el ruido del sensor, que sale de crypto/rand. Devuelve Width·Height·3 bytes.
func Frame(t float64) []byte {
	img := make([]byte, Width*Height*3)
	noise := make([]byte, len(img))
	rand.Read(noise)
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			fx, fy := float64(x)/Width, float64(y)/Height
			lava := 0.5 + 0.25*math.Sin(9*fx+t) + 0.25*math.Cos(7*fy-1.3*t)
			i := (y*Width + x) * 3
			for c := 0; c < 3; c++ {
				v := lava*200 + float64(noise[i+c]%32) // ruido del sensor: 5 bits por canal
				img[i+c] = byte(min(v, 255))
			}
		}
	}
	return img
}

// Local lee n bytes del generador criptográfico del sistema.
func Local(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}
