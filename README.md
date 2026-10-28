# tech-autopsy · el laboratorio

Código de los videos de **Tech Autopsy** ([@tech.autopsy](https://www.tiktok.com/@tech.autopsy)): cada caso real se analiza en video y aquí queda una demo o herramienta que reproduce el problema y su antídoto. El análisis completo de cada caso está en **[duvanmarin.dev](https://www.duvanmarin.dev)**.

Todo sale de fuentes primarias (post-mortems oficiales y blogs de ingeniería de las propias empresas), enlazadas en cada README.

## Series
- **`autopsia/`**: fallos reales de producción. Qué los causó y cómo evitarlos.
- **`anatomia/`**: arquitecturas que funcionan a gran escala, explicadas por dentro.
- **`diagnostico/`**: decisiones de ingeniería que cambiaron los resultados de una empresa.
- **`boletin/`**: noticias técnicas de la semana (videos de fin de semana), explicadas por dentro.

## Casos
| # | serie | caso | código |
|---|---|---|---|
| 001 | autopsia | GitLab 2017 · `rm -rf` en la base de datos de producción | [go-guard](https://github.com/DesiAPG/tech-autopsy-gitlab-2017) (repo propio) |
| 002 | autopsia | Cloudflare 2019 · una regex tumbó la red | [`autopsia/002-cloudflare-2019-regex`](autopsia/002-cloudflare-2019-regex) · `regex-guard` |
| 003 | anatomia | Cloudflare · LavaRand: lámparas de lava para la criptografía | [`anatomia/003-cloudflare-lavarand`](anatomia/003-cloudflare-lavarand) · `entropy-mix` |
| 004 | diagnostico | Cloudflare · Pingora: tiraron NGINX y escribieron su propio proxy | [`diagnostico/004-cloudflare-pingora`](diagnostico/004-cloudflare-pingora) · `pool-bench` |
| 005 | anatomia | Cloudflare · Quicksilver: un cambio en todo el planeta en segundos | [`anatomia/005-cloudflare-quicksilver`](anatomia/005-cloudflare-quicksilver) · `qs-mini` |
| 006 | autopsia | Cloudflare 2025 · el archivo que duplicó su tamaño | [`autopsia/006-cloudflare-2025-feature-file`](autopsia/006-cloudflare-2025-feature-file) · `config-gate` |
| 007 | autopsia | AWS 2017 · un input mal escrito apagó S3 en us-east-1 | [`autopsia/007-aws-2017-s3`](autopsia/007-aws-2017-s3) · `capacity-guard` |
| 008 | anatomia | Figma · el último que escribe gana: multijugador sin OT | [`anatomia/008-figma-multiplayer`](anatomia/008-figma-multiplayer) · `multiplayer` |
| 009 | diagnostico | Discord · de 177 a 72 nodos: request coalescing delante de la base | [`diagnostico/009-discord-coalescing`](diagnostico/009-discord-coalescing) · `coalesce-bench` |

## Videos largos (YouTube)
| # | caso | código |
|---|---|---|
| L01 | CrowdStrike 2024 · faltaba 1 casilla: el Channel File 291 | [`autopsia/L01-crowdstrike-2024`](autopsia/L01-crowdstrike-2024) · `content-gate` |

## Boletines
| # | noticia | código |
|---|---|---|
| B01 | PHP2Shell · dos fallas críticas en el núcleo de PHP (CVE-2026-109618 y 109619) | [`boletin/b01-php-patch-check`](boletin/b01-php-patch-check) · `php-patch-check` |
| B02 | Anthropic · Scion: por qué reemplazó Git en su monorepo | [`boletin/b02-content-addressed`](boletin/b02-content-addressed) · almacén direccionado por contenido |

## Uso
Un solo módulo de Go (1.24+), sin dependencias externas:
```bash
git clone https://github.com/DesiAPG/tech-autopsy && cd tech-autopsy
go test ./...
go run ./autopsia/002-cloudflare-2019-regex/cmd/regex-guard bench
go run ./anatomia/003-cloudflare-lavarand/cmd/entropy-mix demo
go run ./diagnostico/004-cloudflare-pingora/cmd/pool-bench
go run ./anatomia/005-cloudflare-quicksilver/cmd/qs-mini demo
go run ./anatomia/008-figma-multiplayer/cmd/multiplayer
go run ./diagnostico/009-discord-coalescing/cmd/coalesce-bench
go run ./autopsia/006-cloudflare-2025-feature-file/cmd/config-gate check autopsia/006-cloudflare-2025-feature-file/testdata/v42.json -prev autopsia/006-cloudflare-2025-feature-file/testdata/v41.json
go run ./autopsia/007-aws-2017-s3/cmd/capacity-guard --match 's3-*' --dry-run
go run ./autopsia/L01-crowdstrike-2024/cmd/content-gate demo
go run ./boletin/b01-php-patch-check --version 8.4.27
go run ./boletin/b02-content-addressed checkout main
```

Licencia MIT · Duván Marín
