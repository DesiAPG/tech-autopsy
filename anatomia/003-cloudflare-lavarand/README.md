# Caso 003 · LavaRand: lámparas de lava para la criptografía (Cloudflare)

> **Anatomía.** Video: *Lámparas de lava que protegen internet* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## Cómo funciona
En el lobby de la oficina de Cloudflare en San Francisco hay una pared de lámparas de lava. No es decoración: es **una fuente extra de entropía** para sus máquinas de producción (no la única).

1. Una cámara fotografía la pared. La entropía sale del movimiento de la lava y del ruido del sensor de la cámara.
2. Aun suponiendo una foto de 100×100 píxeles y un atacante que acierta cada píxel con solo un bit de error, quedan 100 × 100 × 3 = **30 000 bits**: "órdenes de magnitud más de lo que necesitamos".
3. Un servidor de la oficina mezcla la imagen con su propia entropía; un servicio en un centro de datos vuelve a mezclar.
4. Al arrancar, cada máquina de producción pide un bloque a ese servicio por TLS y lo escribe en `/dev/random`.
5. **El huevo y la gallina:** para hacer TLS ya hace falta aleatoriedad. Lo resuelven con una semilla secreta propia de cada máquina más un HMAC de la hora en nanosegundos, que siembra un CSPRNG solo para esa conexión.

La clave es la mezcla, `H(x, y, z)`: mientras **una sola** fuente siga siendo secreta, el resultado es impredecible. El único ataque fatal es ejecutar código en el servicio de producción que genera el resultado final. Hoy suman péndulos dobles en Londres y móviles colgantes en Austin.

Fuentes oficiales:
- [LavaRand in Production: The Nitty-Gritty Technical Details](https://blog.cloudflare.com/lavarand-in-production-the-nitty-gritty-technical-details/), Joshua Liebow-Feeser, 2017-11-06.
- [Harnessing chaos in Cloudflare offices](https://blog.cloudflare.com/harnessing-office-chaos/), 2024.

## La demo: `entropy-mix`
```bash
go install github.com/DesiAPG/tech-autopsy/anatomia/003-cloudflare-lavarand/cmd/entropy-mix@latest

entropy-mix demo                 # mezcla 3 fuentes; el atacante conoce 2
entropy-mix lint archivo.go ...  # math/rand usado para tokens o claves; sale con 1 si hay hallazgos
```

`demo` mezcla tres fuentes con **HKDF-SHA256**: una "cámara" simulada (foto de 100×100 RGB con ruido de sensor), una fuente **comprometida** que controla el atacante y 32 bytes de `crypto/rand`. El atacante tiene la foto y la fuente comprometida, pero no la local:

```
$ entropy-mix demo
fuente        bytes   ¿el atacante la tiene?
cámara        30000   sí · robó la foto
comprometida     32   sí · la controla
local            32   no
mezcla HKDF-SHA256 → 5025…40d6
atacante (2 de 3)  → 5add…e828  ✖ falla
atacante (3 de 3)  → 5025…40d6  ✓ igual
bits distintos: 118/256
basta UNA fuente secreta ✓
```

Con dos de tres fuentes, la predicción del atacante difiere en unos 128 de 256 bits, que es lo mismo que adivinar al azar. Solo acierta si las compromete todas. (Los valores cambian en cada ejecución.)

`lint` busca `math/rand` (o `math/rand/v2`) en funciones o variables con nombre de secreto (`token`, `key`, `secret`, `session`, `otp`, `nonce`…):

```
$ entropy-mix lint testdata/tokens.go.txt
testdata/tokens.go.txt:13:15: ✖ math-rand-para-secretos · rand.Intn en "NewSessionToken"
testdata/tokens.go.txt:19:9: ✖ math-rand-para-secretos · rand.Intn en "otp"
2 hallazgo(s). Usa crypto/rand
```

### Límites (honestidad ante todo)
- La "cámara" es una simulación: su ruido sale de `crypto/rand`. La demo enseña **la mezcla**, no reemplaza al CSPRNG del sistema.
- En tu código no hace falta mezclar nada a mano: `crypto/rand` en Go, `crypto.getRandomValues()` en el navegador o `/dev/urandom` ya mezclan varias fuentes en el kernel.
- `lint` es una heurística por nombres: no sigue los valores entre funciones y puede tener falsos positivos (`monkey` contiene `key`).

## La lección
Nunca dependas de una sola fuente de aleatoriedad: mezcla fuentes independientes y basta con que una siga siendo secreta. Y nunca uses `Math.random` para secretos.
