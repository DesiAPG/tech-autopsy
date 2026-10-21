# Caso 006 · El archivo que tumbó Cloudflare (2025-11-18)

> **Autopsia.** Video: *El archivo que tumbó Cloudflare* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## Qué pasó
El 18 de noviembre de 2025, a las 11:20 UTC, Cloudflare tuvo su peor caída desde 2019. No fue un ataque: fue un **archivo de configuración que duplicó su tamaño**.

1. El sistema contra bots usa un modelo de machine learning que lee un **archivo de features**, regenerado cada 5 minutos con una consulta a ClickHouse y publicado a toda la red.
2. La consulta (`SELECT name, type FROM system.columns WHERE table = 'http_requests_features'`) **no filtraba por base de datos**. A las 11:05 un cambio de permisos hizo visibles las tablas de la base `r0`, y la consulta empezó a devolver cada columna dos veces.
3. El cambio se aplicaba al clúster por partes: según el nodo que respondiera, salía el archivo bueno o el malo. Por eso la red caía y se recuperaba cada 5 minutos, y al principio parecía un DDoS.
4. El proxy nuevo, **FL2** (Rust), preasigna memoria para **200 features**. Con más, el código devolvía un error… y había un `unwrap()`: `thread fl2_worker_thread panicked: called Result::unwrap() on an Err value`. Cada petición terminaba en 5xx.
5. 14:30: archivo bueno en todo el mundo. 17:06: todo normal.

El archivo malo viajó por **Quicksilver** ([caso 005](../../anatomia/005-cloudflare-quicksilver)): la misma velocidad que hace grande a Cloudflare. La lección no es "no uses `unwrap()`", es **no confiar ciegamente en la configuración que generan tus propios sistemas**. Su respuesta, *Code Orange*: despliegues por etapas también para la configuración, valores seguros por defecto en vez de pánico y quitar dependencias circulares de las herramientas de emergencia.

Fuentes oficiales:
- [Cloudflare outage on November 18, 2025](https://blog.cloudflare.com/18-november-2025-outage/), Matthew Prince, 2025-11-18.
- [Code Orange: Fail Small](https://blog.cloudflare.com/fail-small-resilience-plan/), 2025-12-19.

## La demo: `config-gate`
```bash
go install github.com/DesiAPG/tech-autopsy/autopsia/006-cloudflare-2025-feature-file/cmd/config-gate@latest
config-gate check NUEVO.json -prev ULTIMO_BUENO.json [-max 200] [-growth 1.5]
```

Antes de propagar un archivo generado, valida el **esquema**, el **límite** que soporta el consumidor, los **duplicados** y el **crecimiento** contra la última versión buena. Si algo falla, sale con código 1, no lo publica y se queda con la versión anterior (*fail small*).

`testdata/` reproduce el caso: `v41.json` trae 60 features; `v42.json` es lo que devuelve la consulta sin filtro de base, cada columna dos veces.

```
$ config-gate check testdata/v42.json -prev testdata/v41.json
v42 · 120 features (antes v41 · 60)
esquema      ✓ name y type en todas
límite       ✓ 120 / 200
duplicados   ✖ 60
crecimiento  ✖ ×2.0 (máx ×1.5)
BLOQUEADO · se mantiene v41 · no se propaga
```

Fíjate en el límite: **120 de 200 pasa**. Un límite solo te avisa cuando ya es tarde; los duplicados y el crecimiento contra la versión anterior frenan el archivo malo antes.
