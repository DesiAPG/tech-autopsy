# Caso 009 · De 177 a 72 nodos (Discord)

> **Diagnóstico.** Video: *Discord, de 177 a 72 nodos* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## La decisión
A inicios de 2022, Discord guardaba billones de mensajes en un clúster de **177 nodos de Cassandra**. El dolor eran las **particiones calientes**: cada mensaje se guarda por canal y por una ventana fija de tiempo, así que cuando un servidor enorme publica un anuncio, miles de personas leen la misma partición a la vez, esos nodos se saturan y la latencia sube en todo el clúster.

Migraron a **ScyllaDB** (compatible con Cassandra, en C++, sin recolector de basura, un shard por núcleo), pero la pieza clave fue otra: unos **servicios de datos en Rust** delante de la base. Si muchos usuarios piden la misma fila al mismo tiempo, la base recibe **una sola consulta** y todos comparten la respuesta (*request coalescing*). Enrutan por ID de canal para que las peticiones iguales caigan en la misma instancia.

Resultado: **72 nodos de ScyllaDB**; leer el historial pasó de un p99 de 40–125 ms a **15 ms**, e insertar de 5–70 ms a **5 ms** estables.

La lección: el gran salto no vino de comprar más máquinas. Vino de proteger la base de su patrón de acceso.

Fuente oficial: [How Discord Stores Trillions of Messages](https://discord.com/blog/how-discord-stores-trillions-of-messages), Bo Ingram, 2023-03-06.

## La demo: `coalesce-bench`
```bash
go install github.com/DesiAPG/tech-autopsy/diagnostico/009-discord-coalescing/cmd/coalesce-bench@latest
coalesce-bench                 # 1000 peticiones iguales a la vez
coalesce-bench -n 5000 -costo 10ms
```

Lanza N goroutines que piden el mismo canal a la vez contra una "base" simulada que se vuelve más lenta cuantas más consultas tiene en vuelo (como una partición caliente). Primero sin agrupar, luego con un **singleflight mínimo** escrito con la librería estándar (la misma idea que [`golang.org/x/sync/singleflight`](https://pkg.go.dev/golang.org/x/sync/singleflight)): la primera petición consulta y las que llegan mientras está en vuelo esperan y comparten la respuesta.

```
$ coalesce-bench
1000 peticiones al mismo canal · consulta base 5ms
modo              consultas        p99
sin coalescing         1000      104ms
con singleflight          1        5ms
```

Singleflight solo agrupa peticiones que están en vuelo al mismo tiempo dentro de un proceso; por eso Discord enruta por canal, para que las peticiones iguales lleguen a la misma instancia. (Las cifras cambian según la máquina.)
