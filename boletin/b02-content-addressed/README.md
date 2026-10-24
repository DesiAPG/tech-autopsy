# Boletín B02 · Por qué Anthropic creó Scion (2026-10-13)

> **Boletín de fin de semana.** Video: *Por qué Anthropic creó Scion* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## Qué pasó
Anthropic contó en su blog de ingeniería que reemplazó Git en su monorepo por **Scion**, un sistema de control de versiones propio. En mayo de 2026 mergeaban unos **2.500** cambios al día; en septiembre, **más de 30.000**. Un equipo de **siete personas** lo construyó y migró el monorepo en **cinco meses**.

1. **El choque:** Git asume clientes offline con una copia completa del repositorio. Casi todos sus clientes son agentes y jobs de CI, siempre conectados: **25 millones** de checkouts al día, con una copia de trabajo comprimida de **más de 25 GB**.
2. **Changesets:** en vez de un commit que apunta a un árbol completo, guardan solo la lista de archivos que cambiaron, con el hash de su contenido.
3. **Todo en el servidor:** commit, merge y rebase pasan en el servidor. No hay push.
4. **Inmutable = cacheable:** los objetos se guardan por su hash y nunca cambian, así que cualquier caché los sirve sin coordinarse. Tres niveles (daemon por máquina, caché regional, servidor), cada uno con más del 99 % de aciertos.
5. **Checkout al revés:** el cliente no dice lo que tiene; pide lo que quiere, desde el árbol raíz hacia abajo.

Resultado según Anthropic: checkouts de CI en segundos en vez de minutos y una infraestructura 200 veces más pequeña en CPU y memoria que sus espejos de Git. Y lo más importante: **Scion no es un reemplazo de Git para uso general** (lo dicen ellos) y no se va a publicar.

Fuente oficial: [Redesigning source control at Anthropic](https://www.anthropic.com/engineering/redesigning-source-control), 2026-10-13.

## La demo: un almacén direccionado por contenido
No es Scion ni imita su API: es el concepto en unas 300 líneas de Go, sin dependencias.

```bash
go run ./boletin/b02-content-addressed checkout main          # checkout completo
go run ./boletin/b02-content-addressed checkout main docs     # checkout sparse
```

```
$ go run . checkout main

agente A (ya tenía la versión anterior) · 403 archivos en el checkout
  objetos pedidos: solo los que faltan · 8 de 419
  bytes bajados ≪ clon completo · 17113 B vs 380015 B

agente B (vacío, misma máquina) · 403 archivos
  aciertos de caché ✓ (inmutables) · 411 de 411 desde scion-daemon (máquina)

lecturas al servidor en total: 419 · el resto lo respondieron las cachés
```

- `Server.Commit` aplica un **changeset** en el servidor y mueve la rama: el cliente nunca hace push.
- Cada objeto (archivo o árbol) se guarda por su **SHA-256**. El cliente verifica el hash al recibirlo.
- `Cache` es un LRU que se puede apilar (daemon → regional → servidor). Nunca invalida, porque nada cambia.
- `Client.Checkout` baja la raíz y solo los árboles y archivos que no tiene. Con rutas, hace un checkout sparse.

**Para tu equipo:** no necesitas un Scion. Si tu CI clona el repo completo en cada job, prueba antes `git clone --filter=blob:none`, `git sparse-checkout` y una caché de objetos en los runners.
