# Caso 002 · Una regex tumbó Cloudflare (2019-07-02)

> **Autopsia.** Video: *Una regex tumbó Cloudflare* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## Qué pasó
El 2 de julio de 2019, a las 13:42 UTC, Cloudflare desplegó una regla nueva en su WAF para detectar XSS. Pasó la revisión y los tests de CI, y en segundos llegó a todas sus máquinas. La regla contenía esta regex:

```
(?:(?:\"|'|\]|\}|\\|\d|(?:nan|infinity|true|false|null|undefined|symbol|math)|\`|\-|\+)+[)]*;?((?:\s|-|~|!|{}|\|\||\+)*.*(?:.*=.*)))
```

La parte crítica es `.*(?:.*=.*)`, que se reduce a `.*.*=.*`. El WAF usaba PCRE, un motor con **backtracking** y sin límite de complejidad. La CPU de todos los núcleos que sirven HTTP/HTTPS se fue casi al 100 % en todo el mundo, y cualquier sitio detrás de Cloudflare devolvió **502** durante **27 minutos**.

Causas que convergieron, según el post-mortem:
- una regex que podía hacer backtracking enorme;
- una protección contra el exceso de CPU **eliminada por error** en una refactorización semanas antes;
- un motor de regex sin garantías de complejidad;
- tests que no medían el consumo de CPU;
- reglas del WAF desplegadas a todo el mundo **sin etapas** (el resto del software pasa por DOG → PIG → Canary).

Fuente oficial: [Details of the Cloudflare outage on July 2, 2019](https://blog.cloudflare.com/details-of-the-cloudflare-outage-on-july-2-2019/), John Graham-Cumming, 2019-07-12.

## Por qué explota
Con `.*.*=.*` y un texto sin `=`, el motor prueba **cada forma de repartir el texto** entre los dos `.*`, en cada posición de inicio, antes de rendirse. Un motor de tiempo lineal (Thompson, 1968: RE2, `regexp` de Go, `regex` de Rust) recorre cada carácter una sola vez.

`regex-guard bench` lo mide en tu máquina con un motor de backtracking mínimo frente a `regexp` de Go:

```
       n   pasos backtrack    backtracking    regexp de Go
      25              3653            10µs             7µs
     100            182103           463µs             2µs
     400          10908403        27.844ms            11µs
    1600         686513603       1.652526s            43µs
```

Cuatro veces más texto (400 → 1 600 caracteres): el backtracking tarda unas **60 veces más**, y el motor lineal unas 4 veces. Los pasos no coinciden con los del post-mortem, que usa otro depurador y cuenta distinto, pero el crecimiento es el mismo fenómeno.

## La herramienta: `regex-guard`
Un linter para tu CI que revisa tus regex antes de que lleguen a producción.

```bash
go install github.com/DesiAPG/tech-autopsy/autopsia/002-cloudflare-2019-regex/cmd/regex-guard@latest

regex-guard lint reglas.txt              # un patrón por línea (# = comentario); sale con 1 si hay hallazgos
regex-guard lint -e '.*(?:.*=.*)'
regex-guard bench                        # la tabla de arriba, en tu máquina
```

```
$ regex-guard lint testdata/reglas.txt
testdata/reglas.txt:3: ✖ comodines-seguidos · dos comodines sin límite seguidos…
    en: .* … .*
testdata/reglas.txt:8: ✖ cuantificador-anidado · cuantificador sin límite dentro de otro…
    en: ([0-9A-Z_a-z]+[\t\n\f\r ]?)*
```

En GitHub Actions:
```yaml
- run: go install github.com/DesiAPG/tech-autopsy/autopsia/002-cloudflare-2019-regex/cmd/regex-guard@latest
- run: regex-guard lint config/waf-rules.txt
```

### Qué detecta
| regla | ejemplo | por qué |
|---|---|---|
| `comodines-seguidos` | `.*.*=.*`, `.*(?:.*=.*)`, `a.*\s*.+b` | dos comodines sin límite compiten por los mismos caracteres (el caso Cloudflare) |
| `cuantificador-anidado` | `(a+)+`, `(?:a*b*)*`, `^(\w+\s?)*$` | repeticiones dentro de repeticiones: crecimiento exponencial (ReDoS clásico) |

### Límites (honestidad ante todo)
- Es una **heurística**: puede dejar pasar patrones peligrosos más sutiles, como alternativas que se solapan (`(a|aa)+`).
- Usa el parser de Go: los patrones con lookarounds o backreferences se reportan como "no analizable" y no se revisan.
- **La defensa de verdad es un motor lineal.** Si procesas input no confiable, usa RE2 o el motor de tu lenguaje con tiempo lineal, o como mínimo un timeout por evaluación. regex-guard es una red de seguridad para el CI.

## La lección
El error no fue la regex: fue que un solo cambio pudiera llegar a todo el planeta sin pasar por un canario. Cloudflare reintrodujo la protección de CPU, revisó a mano sus 3 868 reglas, agregó perfilado de rendimiento a los tests, migró a un motor con garantías (RE2 o el de Rust) y llevó el despliegue por etapas a las reglas.
