// deploy-verify: antes de encender una feature flag, comprueba que el deploy llegó a TODOS los nodos;
// y un kill switch que corta el envío si un nodo manda muchas más órdenes de las que recibe (Knight Capital, 2012).
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
)

type nodo struct {
	nombre  string
	binario []byte // lo que realmente corre en el nodo
}

func hash(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])[:8]
}

// verificar devuelve los nodos cuyo binario no coincide con el esperado.
func verificar(nodos []nodo, esperado string) []string {
	var malos []string
	for _, n := range nodos {
		if hash(n.binario) != esperado {
			malos = append(malos, n.nombre)
		}
	}
	return malos
}

// killSwitch corta un nodo si sus órdenes hijas superan max veces las órdenes padre recibidas.
type killSwitch struct {
	max           int
	padres, hijas int
	cortado       bool
}

func (k *killSwitch) padre() { k.padres++ }

func (k *killSwitch) hija() bool {
	if k.cortado {
		return false
	}
	k.hijas++
	if k.padres > 0 && k.hijas > k.max*k.padres {
		k.cortado = true
		return false
	}
	return true
}

func main() {
	viejo := flag.Int("viejo", 8, "nodo que se quedó con el código viejo (0 = ninguno)")
	max := flag.Int("max", 10, "máximo de órdenes hijas por orden padre antes de cortar")
	flag.Parse()

	nuevo := []byte("SMARS v2012.08 · RLP")
	powerPeg := []byte("SMARS v2003 · Power Peg")
	var nodos []nodo
	for i := 1; i <= 8; i++ {
		b := nuevo
		if i == *viejo {
			b = powerPeg
		}
		nodos = append(nodos, nodo{fmt.Sprintf("node-%d", i), b})
	}

	esperado := hash(nuevo)
	fmt.Printf("deploy esperado: %s · 8 nodos\n", esperado)
	for _, n := range nodos {
		estado := "ok"
		if hash(n.binario) != esperado {
			estado = "DISTINTO"
		}
		fmt.Printf("%-7s %s  %s\n", n.nombre, hash(n.binario), estado)
	}
	if malos := verificar(nodos, esperado); len(malos) > 0 {
		fmt.Printf("flag RLP: BLOQUEADA (%d nodo sin deploy)\n", len(malos))
	} else {
		fmt.Println("flag RLP: activada en los 8 nodos")
	}

	// Si aun así se activa, el nodo viejo (sin contador) manda hijas sin parar: el kill switch lo corta.
	ks := &killSwitch{max: *max}
	enviadas := 0
	for p := 0; p < 212; p++ {
		ks.padre()
		for h := 0; h < 1000 && ks.hija(); h++ {
			enviadas++
		}
		if ks.cortado {
			break
		}
	}
	fmt.Printf("kill switch: node-%d cortado tras %d hijas\n", *viejo, enviadas)
}
