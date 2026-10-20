# tech-autopsy · el laboratorio

Código de los videos de **Tech Autopsy** ([@tech.autopsy](https://www.tiktok.com/@tech.autopsy)): cada caso real se analiza en video y aquí queda una demo o herramienta que reproduce el problema y su antídoto. El análisis completo de cada caso está en **[duvanmarin.dev](https://www.duvanmarin.dev)**.

Todo sale de fuentes primarias (post-mortems oficiales y blogs de ingeniería de las propias empresas), enlazadas en cada README.

## Series
- **`autopsia/`**: fallos reales de producción. Qué los causó y cómo evitarlos.
- **`anatomia/`**: arquitecturas que funcionan a gran escala, explicadas por dentro.
- **`diagnostico/`**: decisiones de ingeniería que cambiaron los resultados de una empresa.

## Casos
| # | serie | caso | código |
|---|---|---|---|
| 001 | autopsia | GitLab 2017 · `rm -rf` en la base de datos de producción | [go-guard](https://github.com/DesiAPG/tech-autopsy-gitlab-2017) (repo propio) |
| 002 | autopsia | Cloudflare 2019 · una regex tumbó la red | [`autopsia/002-cloudflare-2019-regex`](autopsia/002-cloudflare-2019-regex) · `regex-guard` |
| 003 | anatomia | Cloudflare · LavaRand: lámparas de lava para la criptografía | [`anatomia/003-cloudflare-lavarand`](anatomia/003-cloudflare-lavarand) · `entropy-mix` |

## Uso
Un solo módulo de Go (1.24+), sin dependencias externas:
```bash
git clone https://github.com/DesiAPG/tech-autopsy && cd tech-autopsy
go test ./...
go run ./autopsia/002-cloudflare-2019-regex/cmd/regex-guard bench
go run ./anatomia/003-cloudflare-lavarand/cmd/entropy-mix demo
```

Licencia MIT · Duván Marín
