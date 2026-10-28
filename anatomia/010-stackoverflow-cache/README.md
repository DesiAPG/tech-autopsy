# Caso 010 · Stack Overflow en un servidor

> **Anatomía.** Video: *Stack Overflow en un servidor* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## La arquitectura
En 2016, toda la red de Stack Exchange servía **209 millones de peticiones HTTP al día** y **66 millones de páginas** con un puñado de máquinas: **11 servidores web**, **4 SQL Server**, **2 Redis**, 3 del motor de etiquetas, 3 de Elasticsearch y 4 balanceadores HAProxy. Según su propio equipo, les bastaba **1 servidor web**: "We have unintentionally tested this, successfully, a few times".

La pieza clave es **no ir a la base de datos**. Una petición entra por HAProxy y llega a un servidor web, que busca primero en su caché local (**L1**, en memoria) y luego en **Redis (L2)**, compartido por todos. Solo si ninguno la tiene, baja a SQL Server, con consultas escritas a mano con **Dapper**. Redis atendía 5.800 millones de hits al día con cada instancia por debajo del 2 % de CPU, y la página de una pregunta se generaba en **22,71 ms** de promedio.

El costo: sin una nube que escale sola, el margen lo pones tú. Por eso todo es redundante (doble red, doble fuente de poder, UPS y generadores). Y el propio autor lo aclara: "I'm saying it works. I'm not saying it's a good idea."

La lección: antes de distribuir, mide. Caché en capas y máquinas grandes llevan más lejos de lo que se cree.

Fuente oficial: [Stack Overflow: The Architecture – 2016 Edition](https://nickcraver.com/blog/2016/02/17/stack-overflow-the-architecture-2016-edition/), Nick Craver, 2016-02-17.

## La demo: `cache-bench`
```bash
go install github.com/DesiAPG/tech-autopsy/anatomia/010-stackoverflow-cache/cmd/cache-bench@latest
cache-bench                       # 20.000 peticiones, 64 en paralelo
cache-bench -sql 10ms -par 128
```

Levanta **un solo servidor HTTP** (`net/http`) y le manda peticiones a preguntas con popularidad tipo Zipf (pocas preguntas muy visitadas, muchas poco visitadas). Lo mide en tres modos:

- **sin caché**: todo va a una "SQL" simulada que se vuelve más lenta cuantas más consultas tiene en vuelo;
- **solo L2**: un almacén compartido con un viaje de red en cada consulta (el papel de Redis);
- **L1 + L2**: primero la memoria del propio servidor, luego L2, y SQL solo si nadie lo tiene.

```
$ cache-bench
1 instancia · 20000 peticiones · 64 en paralelo
modo         req/s     p50    L1    L2   SQL
sin caché     3010 21.25ms    0%    0%  100%
solo L2      18947  1.21ms    0%   86%   14%
L1 + L2      21471    30µs   86%    0%   14%
```

Con una sola instancia, L1 absorbe lo que antes respondía L2; en una granja de servidores, L2 es lo que comparten entre todos. Solo librería estándar: el L2 es un mapa con latencia simulada, no un Redis real. (Las cifras cambian según la máquina; las de Stack Overflow son de su post de 2016.)
