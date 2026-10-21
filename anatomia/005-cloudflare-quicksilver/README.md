# Caso 005 · Quicksilver: un cambio en todo el planeta en segundos (Cloudflare)

> **Anatomía.** Video: *Un cambio en todo el planeta en 2 segundos* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## Cómo funciona
Quicksilver es el sistema con el que Cloudflare lleva cada cambio de configuración (un registro DNS, una regla del WAF) a todas sus máquinas. En 2019, el 99 % de los cambios llegaba a todo el mundo en menos de **2,29 s**; en 2020 cubría 200 ciudades en 90 países.

**Antes** usaban Kyoto Tycoon: funcionó para las primeras 25 ciudades y empezó a romperse pasadas las 100. Las escrituras bloqueaban las lecturas, la base se corrompía al desactivar `fsync` y su replicación basada en timestamps perdía entradas. En 2015 escribieron su propio sistema:

1. **Motor LMDB:** optimizado para leer, varios procesos leyendo a la vez y escrituras solo-añadir; un crash no lo deja corrupto.
2. **Log con número de secuencia:** cada cambio lleva un número exactamente +1. Si a una réplica le llega el 23 y luego el 25, sabe que falta el 24 y lo pide. La replicación se vuelve verificable.
3. **Un solo punto de escritura** y lotes de 500 ms que bajan en abanico: top-mains → mains → cada máquina.

Defensas: un ID único por base, un CRC por clave y un hash incremental del log. Resultado: 2,5 billones de lecturas al día con latencia de microsegundos, más de 90 000 instancias y, en más de 3 años, un solo bug y cero corrupción. El costo: un cambio malo también llega a todo el planeta en segundos.

Fuentes oficiales:
- [Introducing Quicksilver: Configuration Distribution at Internet Scale](https://blog.cloudflare.com/introducing-quicksilver-configuration-distribution-at-internet-scale/), Geoffrey Plouviez, 2020-03-30.
- [Details of the Cloudflare outage on July 2, 2019](https://blog.cloudflare.com/details-of-the-cloudflare-outage-on-july-2-2019/) (p99 de 2,29 s).

## La demo: `qs-mini`
```bash
go install github.com/DesiAPG/tech-autopsy/anatomia/005-cloudflare-quicksilver/cmd/qs-mini@latest
qs-mini demo                       # 10 000 cambios, 3 réplicas, 5 % de mensajes perdidos
qs-mini demo -loss 0.2 -skew 30    # más caos: más pérdida y relojes más desfasados
```

Un único escritor numera cada cambio y los envía en lotes a las réplicas por una red que pierde mensajes. Con **secuencia**, cada réplica detecta el hueco y pide el reenvío. Con **timestamps**, la réplica aplica solo lo más nuevo que lo último que vio: lo que se perdió desaparece, y con relojes desfasados también descarta entradas válidas, sin ningún aviso.

```
$ qs-mini demo
10000 cambios · 3 réplicas · 5 % de mensajes perdidos
secuencia   huecos detectados  1522 · reenviados  1522
            réplicas iguales ✓
timestamps  entradas perdidas  1522 · avisos 0
            réplicas distintas ✖
```

Las mismas 1522 entregas perdidas: con secuencia se recuperan todas; con timestamps nadie se entera.
