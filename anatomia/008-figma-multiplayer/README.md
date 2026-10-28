# Caso 008 · El último que escribe gana (Figma)

> **Anatomía.** Video: *El último que escribe gana* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## La arquitectura
Figma descartó la *Operational Transformation* que popularizó Google Docs (*"unnecessarily complex for our problem space"*) y se quedó con ideas de CRDT, simplificadas gracias a un **servidor central** que decide el orden:

- El documento es un mapa **objeto → propiedad → valor** (`Map<ObjectID, Map<Property, Value>>`).
- **Last-writer-wins por propiedad**: el servidor guarda el último valor que le llega para cada propiedad de cada objeto. Dos personas que cambian propiedades distintas del mismo objeto no chocan.
- El cliente aplica sus cambios al instante e **ignora lo que llega del servidor** para una propiedad mientras su propio cambio no está confirmado: así no hay parpadeos.
- **IDs con el prefijo del cliente**: nadie genera el mismo ID, ni siquiera offline.
- **Fractional indexing**: la posición de una capa es una fracción entre 0 y 1; insertar entre 0,3 y 0,4 es usar 0,35. Nadie renumera.
- **Offline**: al volver, el cliente baja el documento fresco y **reaplica encima sus cambios**. Si alguien tocó la misma propiedad mientras estabas offline, tu valor llega después y gana (ver el paso 5 de la demo).

Fuente oficial: [How Figma's multiplayer technology works](https://www.figma.com/blog/how-figmas-multiplayer-technology-works/), Evan Wallace (cofundador de Figma), 2019-10-16.

## La demo: `multiplayer`
```bash
go run ./anatomia/008-figma-multiplayer/cmd/multiplayer
go test -race ./anatomia/008-figma-multiplayer/
```

Levanta un servidor y dos clientes reales (ANA y BETO) conectados por **WebSocket** (implementación mínima de RFC 6455 con la librería estándar, solo para la demo) y recorre los cinco casos:

```
2 · a la vez, propiedades distintas: ANA mueve (x), BETO pinta (color)
  1:1   color=azul  pos=5  x=340
  ✓ sin conflicto: se quedan los dos cambios

3 · a la vez, la MISMA propiedad: ANA naranja, BETO blanco
  llegó 1 · BETO color = blanco
  llegó 2 · ANA  color = naranja
  ✓ gana el último que llegó al servidor: color = naranja (en los 3)

5 · ANA se queda sin internet y sigue editando
  ANA offline: x=500, color=verde · BETO online: color=azul, y=80
  ANA vuelve: baja el documento fresco y reaplica encima solo sus cambios
  1:1   color=verde  pos=5  x=500  y=80
  ✓ y=80 de BETO se conserva (ANA no la tocó)
  × color=verde: ANA pisó el azul de BETO (los dos tocaron la misma propiedad)
```

En el paso 3 el ganador depende de qué cambio llegue antes al servidor; lo que no cambia es que los tres terminan con el mismo documento.

## Lo que este mini servidor no hace
Figma además rechaza cambios que crean ciclos en el árbol (el padre es una propiedad del hijo), maneja el deshacer y guarda el documento en disco. Aquí solo está el núcleo del modelo de sincronización.
