# Caso 013 · El anillo de Slack

> **Anatomía.** Video: *El anillo de Slack* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## La arquitectura
Slack entrega los mensajes en todo el mundo en **500 ms**, con decenas de millones de clientes conectados. Cada canal (que para Slack puede ser un canal normal, un usuario, un equipo, un archivo o un huddle) vive en un **Channel Server**: un servidor con estado que lo guarda en memoria. En horas pico, uno solo atiende **unos 16 millones de canales**.

1. Al abrir Slack, tu app abre un WebSocket con el **Gateway Server** de tu región, que se suscribe a tus canales.
2. Cuando mandas un mensaje, entra por la API de Webapp y llega a un **Admin Server**.
3. El Admin Server hashea el ID del canal y, con **hashing consistente**, encuentra qué Channel Server es su dueño. No hay una tabla central que consultar.
4. Ese Channel Server manda el mensaje a todos los Gateway Servers del mundo suscritos al canal, y cada uno lo empuja por WebSocket a sus usuarios.
5. Si un Channel Server falla, los gestores del anillo (CHARMs) lo reemplazan: el nuevo está listo **en menos de 20 segundos**, y solo los equipos de ese servidor notan latencia mientras tanto.

El costo: los canales viven en la memoria de un servidor, así que cuando uno cae, esos segundos se sienten. La lección: separar quién guarda los datos (Channel Servers) de quién tiene la conexión (Gateways) deja escalar cada lado por separado, y el anillo reparte sin un directorio central que se vuelva el punto débil.

Fuente oficial: [Real-time Messaging](https://slack.engineering/real-time-messaging/), Sameera Thanguda, Slack Engineering, 2023-04-11.

## La demo: `ring`
```bash
go run github.com/DesiAPG/tech-autopsy/anatomia/013-slack-hashing-consistente/cmd/ring@latest demo
ring demo -canales 1000000 -servidores 20 -cae cs-7
```

Reparte canales entre servidores con un anillo de hashing consistente (**160 nodos virtuales** por servidor, para que el reparto quede parejo), tumba uno y cuenta cuántos canales cambian de dueño. Lo compara con la alternativa ingenua, `hash(canal) % N`, donde cambiar N mueve casi todo:

```
$ ring demo
100000 canales · 6 servidores
cae cs-4:
  anillo    mueve 17,0 % ✓
  hash % N  mueve 83,6 % ✗
```

Con el anillo solo se mueven los canales que eran del servidor caído (~1/6); los tests lo comprueban canal por canal. El paquete (`ring.New`, `Add`, `Remove`, `Owner`) sirve tal cual para repartir claves entre nodos. Solo librería estándar.

Y lo que no es código: con tres servidores, una tabla basta. El anillo vale la pena cuando hay muchos nodos que entran y salen.
