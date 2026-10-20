# Caso 004 · Pingora: Cloudflare tiró NGINX (Cloudflare)

> **Diagnóstico.** Video: *Cloudflare tiró NGINX* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## La decisión
El proxy que conecta a Cloudflare con internet atiende más de un billón de peticiones al día. Durante años fue NGINX. Cloudflare lo reemplazó por **Pingora**, su propio proxy en Rust, y con el mismo tráfico usa **un 70 % menos de CPU y un 67 % menos de memoria**.

Lo que dolía en NGINX:
1. **Carga desbalanceada:** cada petición la atiende un solo worker (un proceso) de principio a fin.
2. **Pools fragmentados (lo más grave):** cada worker tiene su propio pool de conexiones hacia los orígenes. Si la petición cae en un worker y la conexión libre está en otro, no se puede reutilizar: hay que abrir otra, con handshake TCP y TLS completos. Cuantos más workers, peor.
3. **C y Lua:** C no es seguro en memoria; la lógica en Lua es lenta y sin tipos.

Pingora usa **hilos en lugar de procesos**, con work stealing sobre Tokio, y todos comparten **un solo pool**. Resultado: un tercio de las conexiones nuevas por segundo; para un gran cliente la reutilización pasó de 87,1 % a 99,92 %; 434 años de handshakes ahorrados cada día; −5 ms en la mediana del TTFB.

La lección: reescribir casi siempre es un error. Aquí funcionó porque el límite estaba en la **arquitectura** (procesos aislados), no en el código, y porque lo midieron durante años antes de decidir.

Fuentes oficiales:
- [How we built Pingora, the proxy that connects Cloudflare to the Internet](https://blog.cloudflare.com/how-we-built-pingora-the-proxy-that-connects-cloudflare-to-the-internet/), Yuchen Wu y Andrew Hauck, 2022-09-14.
- [Open sourcing Pingora](https://blog.cloudflare.com/pingora-open-source), 2024.

## La demo: `pool-bench`
```bash
go install github.com/DesiAPG/tech-autopsy/diagnostico/004-cloudflare-pingora/cmd/pool-bench@latest
pool-bench                  # 4000 peticiones, 16 en vuelo, 16 workers
pool-bench -n 20000 -c 32   # más carga
```

Levanta un origen HTTPS local, cuenta las conexiones TLS nuevas que acepta y envía el mismo tráfico con tres estrategias: una conexión nueva por petición, **un pool por worker** (como los procesos de NGINX; cada petición cae en un worker al azar) y **un pool compartido** (como los hilos de Pingora):

```
$ pool-bench
4000 peticiones · 16 en vuelo · origen HTTPS local
estrategia       handshakes     reuso       p50       p95
sin reutilizar         4000     0.00%    3.98ms    6.29ms
pool por worker         100    97.50%      50µs     500µs
pool compartido          49    98.78%      50µs     370µs
```

En localhost el handshake cuesta milisegundos; contra un origen real cada uno suma además varios viajes de ida y vuelta por la red. (Las cifras cambian según la máquina.)
