# L01 · Faltaba 1 casilla: CrowdStrike, 19 de julio de 2024

> **Autopsia (video largo).** YouTube: *El error de CrowdStrike explicado* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## Qué pasó
El 19 de julio de 2024, a las 04:09 UTC, CrowdStrike publicó una actualización de contenido de rutina (el **Channel File 291**) para su sensor Falcon en Windows. A las 05:27 UTC la revirtió. En esos 78 minutos, según Microsoft, cayeron unos **8,5 millones** de equipos Windows (menos del 1 % del total). Mac y Linux no se vieron afectados.

1. El sensor tiene un driver en el kernel (`csagent.sys`) que carga muy temprano en el arranque. Las reglas de detección (*Rapid Response Content*) llegan desde la nube como configuración, sin tocar el código.
2. En febrero de 2024 (sensor 7.11) llegó una plantilla nueva para detectar abusos de *named pipes* (IPC). Su definición declaraba **21 campos**, pero el código del sensor le entregaba al intérprete **solo 20**.
3. Durante casi cinco meses nadie lo notó: todas las reglas, y los 12 casos de prueba automática, ponían un **comodín** en el campo 21, y un comodín no se lee.
4. El 19 de julio, una regla nueva puso un criterio real en el campo 21. El *Content Validator* la aprobó porque comparaba contra la definición (21), no contra lo que el sensor entrega (20).
5. En cada equipo, al crearse un named pipe, el intérprete leyó la posición 21 de un arreglo de 20 punteros: lectura fuera de límites en el kernel → pantalla azul. El crash dump publicado muestra `r11 = 0x14` (índice 20) en `csagent!TemplateGetString`.
6. Revertir no bastó: el archivo ya estaba en disco y el driver carga temprano. Había que arrancar en modo seguro y borrar `C-00000291*.sys` a mano (con BitLocker, a veces con la clave de recuperación). Al 29 de julio, ~99 % de los sensores estaban de nuevo en línea.

CrowdStrike lo describe como una confluencia de fallos: el compilador no comparaba campos definidos con entregados, el intérprete no revisaba límites, las pruebas nunca usaron el campo 21 sin comodín y el contenido salía a todos a la vez. Hoy valida el número de entradas, revisa límites en tiempo de ejecución, prueba cada campo y despliega por anillos con canario.

Fuentes oficiales:
- [External Technical Root Cause Analysis — Channel File 291](https://crowdstrike.com/wp-content/uploads/2024/08/Channel-File-291-Incident-Root-Cause-Analysis-08.06.2024.pdf), CrowdStrike, 2024-08-06.
- [Preliminary Post Incident Review](https://www.crowdstrike.com/en-us/blog/falcon-content-update-preliminary-post-incident-report/), CrowdStrike, 2024-07-24.
- [Helping our customers through the CrowdStrike outage](https://blogs.microsoft.com/blog/2024/07/20/helping-our-customers-through-the-crowdstrike-outage/), Microsoft, 2024-07-20.

## La demo: `content-gate`
```bash
go run github.com/DesiAPG/tech-autopsy/autopsia/L01-crowdstrike-2024/cmd/content-gate@latest demo
```

Un mini intérprete de reglas que reproduce el caso: plantilla de 21 campos, sensor que entrega 20, la regla de marzo (comodín en el 21) y la del 19 de julio (criterio real en el 21). Muestra las cuatro capas:

- **`ValidateAgainstDefinition`**: el validador del incidente (compara contra la definición). Aprueba la regla.
- **`ValidateAgainstInputs`**: el corregido (compara contra lo que el código entrega). La frena.
- **`Match`**: intérprete con chequeo de límites; devuelve un error controlado. `UnsafeMatch` imita al del incidente: en Go es un *panic*, en el kernel fue una pantalla azul.
- **`Deploy`**: despliegue por anillos (canario → anillo 1 → anillo 2 → todos) con rollback en cuanto cae un equipo.

```
$ content-gate demo
plantilla IPC: 21 campos definidos · 20 entregados

regla 2024-03-05
  validador viejo  ✓
  validador nuevo  ✓
  intérprete       ✓
  anillos          ✓ llegó a 4 de 4 anillos

regla 2024-07-19
  validador viejo  ✓
  validador nuevo  ✕ usa el campo 21; el sensor entrega 20
  intérprete       ✕ campo 21 (índice 20): índice fuera de límites
  anillos          ✕ cayó en canario (10 equipos) → rollback · 1 de 4 anillos
```

La regla de marzo pasa por todo, igual que en producción. La de julio la deja pasar el validador viejo; el nuevo la frena, el intérprete seguro no se cae y, aunque llegara a salir, el canario la devuelve con 10 equipos afectados en lugar de 8,5 millones. Los valores del evento y los tamaños de los anillos son ilustrativos.
