# Boletín B01 · PHP2Shell: dos fallas críticas en PHP (2026-10-21)

> **Boletín de fin de semana.** Video: *Dos fallas críticas en PHP* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## Qué pasó
El 21 de octubre de 2026, PHP publicó **8.5.12, 8.4.27, 8.3.36 y 8.2.35** para arreglar dos vulnerabilidades en su núcleo, las dos con **CVSS v4.0 9.2**: ejecución remota de código sin autenticación. Afectan a toda instalación que sirve peticiones web (PHP-FPM, Apache con mod_php, php-cgi, FrankenPHP, LiteSpeed). No están en WordPress ni en tu código: están en PHP.

1. **CVE-2026-109618 · RCE vía OOM corruption.** PHP maneja el "out of memory" con `longjmp()`. Si la memoria se agota mientras la petición arranca (por ejemplo, con un multipart lleno de encabezados), quedan datos a medio construir y la limpieza los recorre como si fueran válidos. En mod_php pasa antes de que corra una línea de tu aplicación.
2. **CVE-2026-109619 · RCE vía estado global obsoleto.** Un worker atiende muchas peticiones. Si un error fatal se salta la limpieza, una variable global queda apuntando a memoria ya liberada, y la siguiente petición del mismo worker la usa (use-after-free).

Actualizar es **la única mitigación confiable**. Bajar `error_reporting` o usar `@` no protege. PHP 8.1 y anteriores ya no reciben actualizaciones oficiales.

Fuentes oficiales:
- [php.net · avisos del 20, 21 y 23 de octubre de 2026](https://www.php.net/archive/2026.php)
- [GHSA-w8hw-w9fx-j79w](https://github.com/php/php-src/security/advisories/GHSA-w8hw-w9fx-j79w) (CVE-2026-109618) · [GHSA-6wmf-4rv5-p3rp](https://github.com/php/php-src/security/advisories/GHSA-6wmf-4rv5-p3rp) (CVE-2026-109619)

## La herramienta: `php-patch-check`
Revisa **solo la versión y la configuración**. No contiene ni prueba ningún exploit.

```bash
go run ./boletin/b01-php-patch-check                    # usa el `php` del PATH
go run ./boletin/b01-php-patch-check --version 8.4.27   # o una versión concreta
```

```
$ go run . --version 8.4.27
8.4.27  ✓ parche aplicado

$ go run . --version 8.1.31
8.1.31  ✕ rama sin soporte oficial
   nota: el equipo de PHP publicó un backport mínimo de código fuente para 8.1
   ojo: algunas distribuciones parchan sin cambiar la versión; revisa el aviso de tu proveedor
```

Sale con código 1 si la versión no tiene el parche, así que sirve en CI. Cuando consulta el `php` local, también avisa si siguen activas `fatal_error_backtraces` y `session.upload_progress.enabled`, las dos mitigaciones temporales del aviso.

**Límite:** Debian, Ubuntu, RHEL y otros proveedores a veces parchan sin cambiar el número de versión. Si la herramienta marca ✕ en un paquete de distribución, confirma con el aviso de seguridad de tu proveedor.
