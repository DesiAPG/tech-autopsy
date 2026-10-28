# Boletín B03 · Una regex y GitLab: 10 de 10 (2026-10-21)

> **Boletín de fin de semana.** Video: *Una regex y GitLab: 10 de 10* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## Qué pasó
El 21 de octubre de 2026, GitLab publicó un parche crítico para **CVE-2026-110264** (CVSS 10.0): un desbordamiento de entero al compilar una expresión regular especialmente armada que, bajo ciertas condiciones, podía permitir a un usuario **sin autenticar** ejecutar código en el servidor. Afecta a GitLab CE/EE **autoalojado desde la versión 7.0**; GitLab.com y Dedicated ya estaban parchados.

Nueve días antes, el 12 de octubre, GitLab había corregido **CVE-2026-97229** (también CVSS 10.0), otro desbordamiento de entero en el compilador de regex, esa vez en la configuración de CI/CD. Las dos las reportó el mismo investigador por HackerOne.

**Cómo funciona este tipo de falla (en general; GitLab no publicó el patrón):** una regex se compila a un pequeño programa antes de ejecutarse. El compilador calcula cuánta memoria necesita; si esa cuenta pasa el máximo de un entero, da la vuelta y queda pequeña, se reserva poca memoria y luego se escribe fuera de ella. Las detecciones que publicó GitLab nombran a **Onigmo** (el motor de regex de Ruby) y al render de archivos **Org mode**.

No hay mitigación publicada: hay que actualizar a **19.5.1, 19.4.3, 19.3.6, 19.2.9, 19.1.9, 19.0.10 o 18.11.13**. Las versiones 19.4.3 y 19.3.6 traen migraciones: en una instalación de un solo nodo, GitLab no arranca hasta que terminan.

Fuentes oficiales:
- [GitLab Critical Patch Release 19.5.1 (2026-10-21)](https://docs.gitlab.com/releases/patches/patch-release-gitlab-19-5-1-released/)
- [GitLab Patch Release 19.4.2 (2026-10-12)](https://docs.gitlab.com/releases/patches/patch-release-gitlab-19-4-2-released/)

## La herramienta: `gitlab-patch-check`
Compara **solo la versión**. No contiene ni prueba ningún exploit.

```bash
go install github.com/DesiAPG/tech-autopsy/boletin/b03-gitlab-patch-check@latest
gitlab-patch-check --version 19.4.2-ee
GITLAB_TOKEN=... gitlab-patch-check --url https://gitlab.tu-empresa.com   # lee /api/v4/version
```

```
$ gitlab-patch-check --version 19.4.2-ee
19.4.2  ✕ sin arreglo · actualiza a 19.4.3 o superior
   ojo: este parche trae migraciones; en un solo nodo GitLab no arranca hasta que terminan
   GitLab.com ya está parchado; esto es para instancias autoalojadas (CE/EE)
```

Sale con código 1 si la versión no tiene el arreglo, así sirve en un job de CI o en un cron de monitoreo.
