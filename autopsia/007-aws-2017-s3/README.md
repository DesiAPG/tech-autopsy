# Caso 007 · Un input apagó S3 (AWS, 2017-02-28)

> **Autopsia.** Video: *Un input apagó S3* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## Qué pasó
El 28 de febrero de 2017, en us-east-1 (Virginia del Norte), Amazon S3 dejó de atender peticiones durante unas cuatro horas. No fue un ataque ni un fallo de hardware: fue **un comando de rutina con un input mal escrito**.

1. El equipo de S3 investigaba por qué el sistema de **facturación** iba lento y decidió retirar unos pocos servidores de ese subsistema.
2. 9:37 AM PST: un miembro autorizado del equipo, siguiendo un playbook establecido, ejecutó el comando. Uno de los inputs estaba mal escrito y **se retiraron muchos más servidores de los previstos**.
3. Esos servidores sostenían dos subsistemas: **index** (metadatos y ubicación de todos los objetos de la región; sin él no funciona ningún GET, PUT, LIST ni DELETE) y **placement** (decide dónde guardar los objetos nuevos y depende del index).
4. Los dos necesitaron un **reinicio completo**. Llevaban años sin reiniciarse por completo en las regiones grandes y validar todos los metadatos tardó más de lo esperado. Cayeron también la consola de S3, los lanzamientos nuevos de EC2, los volúmenes EBS que leían de snapshots y Lambda.
5. 13:54 PST: S3 vuelve a la normalidad (≈ 4 h 17 min después del comando). Hasta las 11:37 AWS ni siquiera pudo actualizar su panel de estado, porque su consola dependía de S3.

La lección no es "ten cuidado al escribir". Es que **la herramienta permitía quitar demasiada capacidad, demasiado rápido**. AWS la cambió para retirar capacidad más despacio y negarse a bajar cualquier subsistema de su capacidad mínima, partió el index en celdas más pequeñas (menor radio de impacto) y llevó el panel de estado a varias regiones.

Fuente oficial: [Summary of the Amazon S3 Service Disruption in the Northern Virginia (US-EAST-1) Region](https://aws.amazon.com/message/41926/), AWS, 2017.

## La demo: `capacity-guard`
```bash
go install github.com/DesiAPG/tech-autopsy/autopsia/007-aws-2017-s3/cmd/capacity-guard@latest
capacity-guard --match PATRÓN [--pool pool.json] [--dry-run]
```

Antes de retirar nodos calcula el **radio de impacto**: qué subsistemas pierden capacidad y cuánta. Se niega (sale con código 1, sin tocar nada) si algún subsistema queda **por debajo de su mínimo** o si el lote supera el **máximo por lote** (`maxBatchPct`, 10 % del pool por defecto: retirar despacio). Sin `--pool` usa el pool de ejemplo de `testdata/us-east-1.json`: 12 nodos de facturación, 12 de index y 6 de placement.

El input que se quería escribir, unos pocos servidores de facturación:
```
$ capacity-guard --match 's3-billing-0[1-3]' --dry-run
plan: retirar 3 de 30 nodos (lote máx. 3)
✓ billing: 12 → 9 (mínimo 2)
OK · dry-run: no se retiró nada
```

El input mal escrito, que selecciona todo S3:
```
$ capacity-guard --match 's3-*' --dry-run
plan: retirar 30 de 30 nodos (lote máx. 3)
✕ billing: 12 → 0 (mínimo 2)
✕ index: 12 → 0 (mínimo 8)
✕ placement: 6 → 0 (mínimo 4)
RECHAZADO · no se retiró nada
```

El comando real de AWS no es público; el pool y los patrones son ilustrativos. Lo que importa es el patrón: **la herramienta no confía en el input**, y el límite vive en la herramienta, no en la memoria de quien la usa.
