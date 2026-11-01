# Caso 012 · El tipo de ID equivocado (Atlassian, 2022)

> **Autopsia.** Video: *El tipo de ID equivocado* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## Qué pasó
El 5 de abril de 2022, Atlassian tenía que borrar una app vieja (*Insight – Asset Management*) de los sitios de clientes que la tenían, porque ya venía integrada en Jira Service Management. Por un problema de comunicación entre equipos, al script le llegaron **los IDs de los sitios completos** en lugar de los IDs de la app.

La API de borrado **aceptaba los dos tipos de ID** y asumía que el input era correcto: con el ID de una app borraba la app; con el de un sitio, el sitio entero. No había ninguna advertencia. El script había pasado revisión de pares (centrada en qué endpoint se llamaba) y pruebas en Staging, donde esos IDs ni existían.

Entre las **07:38 y las 08:01 UTC** se borraron **883 sitios de 775 clientes** (Jira, Confluence, Opsgenie, Statuspage…). Había backups y nadie perdió más de **5 minutos** de datos, pero en una arquitectura multi-tenant cada base mezcla a muchos clientes: había que sacar cada sitio del backup pieza por pieza. Para una parte de los clientes, volver tomó **hasta 14 días** (todos restaurados el 18 de abril).

La lección: no fue el ingeniero que corrió el script. Fue una API destructiva que confiaba en cualquier ID y borraba de verdad. Atlassian cambió a **borrados suaves universales**: ningún dato de cliente se borra sin pasar por un borrado suave con retención, despliegue escalonado y rollback probado.

Fuente oficial: [Post-Incident Review on the Atlassian April 2022 outage](https://www.atlassian.com/engineering/post-incident-review-april-2022-outage), Atlassian, 2022-04-29.

## La demo: `softdel`
```bash
go run github.com/DesiAPG/tech-autopsy/autopsia/012-atlassian-2022-soft-delete/cmd/softdel@latest demo
```

Cuatro defensas que le faltaron a esa API:

1. **IDs con tipo**: `AppID` y `SiteID` son tipos distintos; `BorrarApp(SiteID("site-7"))` no compila.
2. **Borrado suave con retención**: borrar solo marca; `RestaurarApp` deshace mientras dure la retención y `Purgar` borra de verdad después.
3. **Confirmación explícita** para borrar un sitio entero (hay que repetir su ID).
4. **Tope de borrados por minuto** para los lotes: 883 borrados seguidos se frenan en 50.

```
$ softdel demo
app insight-42: borrado suave ✓
site-7: falta confirmar el tipo ✗
lote de 883: frenado a 50/min ✗
restaurar insight-42 ✓
```

Solo librería estándar.
