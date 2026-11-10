# Caso 018 · Twitter partió el monolito

> **Diagnóstico.** Video: *Twitter partió el monolito* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## La decisión
El 3 de agosto de 2013, en Japón, Twitter recibió **143.199 tuits en un segundo** (unas 25 veces lo normal) y nadie lo notó. Tres años antes, el Mundial 2010 lo dejaba caído por ratos.

1. **Antes:** una de las instalaciones de Ruby on Rails más grandes del mundo, un monolito con unos 200 ingenieros encima. Midieron: CPU y RAM al límite, la red casi sin usar. Cada servidor de Rails atendía **una petición a la vez**: 200–300 peticiones por segundo por máquina.
2. **La decisión:** reescribir sobre la JVM (ya tenían el buscador en Java, y el streaming y Flock en Scala). Estimaron más de 10× en el mismo hardware.
3. **Partir por sustantivos:** servicios de tuits, timelines y usuarios. Los tuits pasaron de una MySQL con un solo maestro a **T-Bird** (repartidos por hash entre muchas bases) y los IDs a **Snowflake**.
4. **El monstruo nuevo:** cada equipo manejaba las fallas a su manera y los clientes reintentaban sin parar contra servicios lentos (*thundering herds*). La respuesta fue **Finagle**: una sola librería de red para conexiones, fallas y balanceo.
5. **Después:** 10.000–20.000 peticiones por segundo por máquina y entre 5 y 12 veces menos máquinas sirviendo el sitio.

La lección: no fue solo cambiar de lenguaje. Midieron qué recurso se agotaba y cambiaron cómo se atienden las peticiones y cómo se parte el sistema.

Fuente oficial: [New Tweets per second record, and how!](https://blog.x.com/engineering/en_us/a/2013/new-tweets-per-second-record-and-how), Raffi Krikorian, blog de ingeniería de Twitter, 2013-08-16.

## La demo: `fanout`
```bash
go run github.com/DesiAPG/tech-autopsy/diagnostico/018-twitter-fanout/cmd/fanout@latest
```

Dos cosas del post, medidas:

```
$ fanout
una página = tuits (30 ms) + timeline (45 ms) + usuario (60 ms)
  un hilo, en secuencia:  135ms  (se suma)
  futures, en paralelo:   61ms  (el más lento)

1000 peticiones a un servicio lento (90 % de timeouts), hasta 3 reintentos por cliente
  cada cliente reintenta por su cuenta: 3400 intentos  ✗ estampida
  presupuesto de reintentos (20 %):     1209 intentos  ✓
```

`Sequential` es un proceso que atiende una petición a la vez: la latencia de la página es la suma. `Concurrent` pide los servicios en paralelo (como los Futures de Finagle): tarda lo del más lento. `Budget` es un presupuesto de reintentos al estilo del de Finagle: cada petición nueva deposita fichas y cada reintento gasta una, así que los reintentos nunca pasan de ~20 % del tráfico. Solo librería estándar.

Y lo que no es código: antes de reescribir, mide. Si la red está libre y la CPU no, el problema puede ser el modelo de concurrencia, no el lenguaje.
