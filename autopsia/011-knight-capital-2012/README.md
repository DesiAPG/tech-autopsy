# Caso 011 · El servidor sin deploy (Knight Capital, 2012)

> **Autopsia.** Video: *El servidor sin deploy* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## Qué pasó
El 1 de agosto de 2012, Knight Capital (cerca del 10 % del volumen de acciones de EE. UU.) llevó código nuevo a **SMARS**, su router de órdenes, para el programa RLP de la NYSE. SMARS parte cada orden padre en órdenes hijas y un contador corta el envío cuando se completa.

En 2003 Knight dejó de usar una función llamada **Power Peg**, pero el código siguió ahí, invocable. En 2005 movieron el contador de órdenes ejecutadas a otro punto y nunca volvieron a probar Power Peg. En 2012, el código nuevo **reutilizó la bandera** que antes encendía Power Peg.

El deploy fue por etapas en **8 servidores**, y un técnico no copió el código nuevo en uno de ellos. Nadie lo revisó; ningún procedimiento lo exigía. A las 9:30 llegaron órdenes con la bandera encendida: 7 servidores corrieron el código nuevo y **el octavo despertó a Power Peg**, sin contador. 212 órdenes de clientes se convirtieron en **4 millones de ejecuciones en 154 acciones** en unos 45 minutos. Desde las 8:01 el sistema había mandado **97 correos** con "Power Peg disabled", que no eran alertas y nadie leía. Y el "arreglo", desinstalar el código nuevo de los **7 servidores buenos**, lo empeoró: todos pasaron a correr Power Peg.

Resultado: **más de 460 millones de dólares perdidos** y una multa de la SEC de **12 millones**.

La lección: no fue el técnico. Fue un sistema sin revisión de deploys, con código muerto invocable y sin un botón para detener el router.

Fuente oficial: [SEC, In the Matter of Knight Capital Americas LLC](https://www.sec.gov/litigation/admin/2013/34-70694.pdf), Release No. 70694, 16-oct-2013.

## La demo: `deploy-verify`
```bash
go install github.com/DesiAPG/tech-autopsy/autopsia/011-knight-capital-2012/cmd/deploy-verify@latest
deploy-verify                 # 8 nodos, el 8 se quedó con el código viejo
deploy-verify -viejo 0        # deploy completo: la flag se activa
```

Dos controles que le faltaron a Knight:

1. **Verificar el deploy antes de la flag**: compara el hash de lo que corre en cada nodo con el esperado y se niega a encender la feature flag si alguno difiere.
2. **Kill switch de tasa**: si un nodo manda más de N órdenes hijas por cada orden padre recibida, se corta solo.

```
$ deploy-verify
deploy esperado: 7b35aabc · 8 nodos
node-1  7b35aabc  ok
…
node-7  7b35aabc  ok
node-8  9e8a0325  DISTINTO
flag RLP: BLOQUEADA (1 nodo sin deploy)
kill switch: node-8 cortado tras 10 hijas
```

Y la tercera, que no es código: **borra el código muerto y nunca recicles una bandera**. Solo librería estándar.
