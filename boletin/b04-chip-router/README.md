# Boletín B04 · Anthropic sale a bolsa: lo que muestra por dentro (2026-10-26)

> **Boletín de fin de semana.** Video: *Anthropic sale a bolsa* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## Qué pasó
El 26 de octubre de 2026, Anthropic presentó su formulario **S-1** ante la SEC para cotizar en el Nasdaq con el símbolo **ANT** (el precio por acción aún no está definido). El documento muestra:

- Ingresos de **$41 M** (2023), **$386 M** (2024), **$4.6 B** (2025) y **$16.3 B** solo en el primer semestre de 2026; pérdida operativa del semestre de **$693 M** (vs $2.1 B un año antes).
- Que la empresa "has consistently operated in a **compute-constrained** environment where customer demand exceeds available compute". El gasto en cómputo e infraestructura pasó de **$944 M** (1T 2025) a **$5.6 B** (2T 2026).
- Que entrena y sirve Claude en **cuatro familias de chips** (AWS Trainium, Google TPU, GPU de NVIDIA y de AMD) con una **capa de orquestación propia** que reparte el trabajo por precio y rendimiento, lo que "expands our available compute, reduces dependence on any single vendor".
- Contratos de cómputo y centros de datos con pagos mínimos futuros de unos **$417 B**, con plazos de menos de 1 año a 20 años.

Fuente oficial: [Anthropic, PBC · Form S-1 (SEC EDGAR)](https://www.sec.gov/Archives/edgar/data/2130357/000162828026067168/anthropicpbc-sx1.htm).

## La demo: `chip-router`
Un planificador mínimo que reparte trabajos entre **flotas de aceleradores distintas** (cada una con su precio, velocidad y número de slots) y lo compara con mandar todo a **una sola flota**. Para cada trabajo elige la flota que minimiza costo + espera ponderada.

```bash
go install github.com/DesiAPG/tech-autopsy/boletin/b04-chip-router@latest
chip-router                  # 20.000 trabajos, 9 por tick
chip-router -ritmo 6         # con menos carga, una sola flota alcanza
```

```
$ chip-router
20000 trabajos · 9 por tick · valores simulados
modo                 costo   espera      p95
1 sola flota        120472    391.5    745.1
4 flotas            104851      0.1      0.6
```

Con una sola flota la demanda supera la capacidad y la cola crece sin parar; repartiendo entre cuatro, la espera desaparece y el costo baja porque las flotas más baratas absorben lo que pueden. **Los valores son simulados**: no representan precios ni rendimiento reales de ningún proveedor. Solo librería estándar.
