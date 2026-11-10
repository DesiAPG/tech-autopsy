# Caso 017 · Los relojes atómicos de Google

> **Anatomía.** Video: *Los relojes atómicos de Google* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## La arquitectura
Spanner es la base de datos global de Google. Para que una transacción que terminó antes que otra tenga siempre la marca de tiempo menor, en cualquier centro de datos del planeta, usa **TrueTime**:

1. `TT.now()` no devuelve una hora: devuelve un intervalo `[earliest, latest]` que garantiza contener la hora real.
2. Cada centro de datos tiene **maestros de tiempo**: la mayoría con receptor GPS y antena propia, y el resto con relojes atómicos (los *Armageddon masters*). Usan los dos porque fallan distinto.
3. Cada máquina consulta a varios maestros **cada 30 segundos** y descarta a los que mienten (algoritmo de Marzullo). La incertidumbre ε suele ir de **1 a 7 ms**.
4. **Commit wait:** al confirmar, el líder le pone a la transacción `s >= TT.now().latest` y **espera a propósito** hasta que `TT.after(s)` sea cierto. En promedio espera al menos 2ε, en paralelo con Paxos.
5. Resultado: si T1 termina antes de que empiece T2, `s1 < s2`. Google lo explica con un banco: el pago nunca queda antes del depósito y nadie paga una multa por un reloj atrasado.

El costo: cada escritura paga milisegundos de espera, y si ε crece (el 13 de abril del paper, al apagar 2 maestros por mantenimiento, durante cerca de una hora), Spanner se frena.

Fuentes oficiales: [Spanner: Google's Globally-Distributed Database](https://research.google/pubs/spanner-googles-globally-distributed-database-2/) (OSDI 2012) · [Spanner: TrueTime and external consistency](https://docs.cloud.google.com/spanner/docs/true-time-external-consistency) (Google Cloud) · [Spanner, TrueTime & The CAP Theorem](https://static.googleusercontent.com/media/research.google.com/en//pubs/archive/45855.pdf) (Eric Brewer, 2017).

## La demo: `truetime`
```bash
go run github.com/DesiAPG/tech-autopsy/anatomia/017-spanner-truetime/cmd/truetime@latest demo
truetime demo -atraso 3ms -eps 4ms
```

El ejemplo del banco en tiempo simulado: dos máquinas, la del pago con el reloj atrasado. Con marcas ingenuas (el reloj local de cada máquina) el pago queda antes que el depósito y una lectura ve el total en negativo. Con TrueTime + commit wait, el orden real se respeta:

```
$ truetime demo
ahorros $50 · corriente $50 · el reloj del pago atrasa 3ms · ε = 4ms

marcas ingenuas (un solo reloj)
  depósito +200  marca 100ms
  pago −150      marca 98ms
  una lectura ve el pago sin el depósito: total $-50 → MULTA ✗

TrueTime + commit wait
  depósito +200  marca 104ms
  pago −150      marca 110.1ms
  ninguna lectura queda en negativo (peor total $150) ✓

espera a propósito: depósito 8.1ms · pago 8.1ms (al menos ~2ε)
```

Los tests prueban la propiedad con 5.000 casos aleatorios: si cada reloj respeta su cota (`|skew| <= ε`), commit wait nunca invierte el orden; sin esperar, sí pasa. El paquete (`Node.Now`, `Node.After`, `CommitWait`) es la regla de Spanner en pocas líneas. Solo librería estándar.

Y lo que no es código: la garantía depende de que ε sea una cota real. Sin hardware de tiempo que la sostenga, usa relojes lógicos (Lamport, HLC) en vez de fingir que tu reloj es exacto.
