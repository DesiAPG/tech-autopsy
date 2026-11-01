# L02 · Medio segundo: la puerta trasera de xz (2024)

> **Autopsia (video largo).** YouTube: *La puerta trasera de xz: el ataque de 2 años que casi abre Linux* · [@tech.autopsy](https://www.tiktok.com/@tech.autopsy) · análisis en [duvanmarin.dev](https://www.duvanmarin.dev)

## Qué pasó
El 29 de marzo de 2024, Andres Freund, desarrollador de PostgreSQL, publicó en la lista oss-security que el paquete xz/liblzma traía una puerta trasera (**CVE-2024-3094**, CVSS 10.0). Lo notó porque un login fallido por SSH tardaba **0,807 s** en lugar de **0,299 s**: medio segundo de más.

1. Desde octubre de 2021, una cuenta llamada **Jia Tan** mandó parches pequeños y útiles a xz, un proyecto que mantenía una sola persona, Lasse Collin, en su tiempo libre. Otras cuentas presionaron en público por un nuevo mantenedor. Para 2023, Jia Tan publicaba las versiones.
2. El 23 de febrero de 2024 entró un commit "Tests: Add a few test files." con dos binarios "corruptos a propósito". Según Freund, en la 5.6.0 ni siquiera se usaban en ninguna prueba.
3. El código en git estaba limpio. La pieza que activaba todo vivía **solo en el tarball**: una línea de más en `m4/build-to-host.m4`, que al configurar la compilación sacaba código ya compilado de esos archivos de prueba y lo metía en liblzma. Solo se activaba en Linux x86-64, con gcc y al construir paquetes Debian o RPM.
4. Debian y otras distribuciones parchean OpenSSH para avisar a systemd; así `sshd` termina cargando libsystemd, y libsystemd carga liblzma. Al arrancar, el código escondido usaba ifunc para desviar la verificación de firmas RSA de sshd. Según Red Hat, interfiere con la autenticación de sshd y parece pensado para permitir acceso no autorizado.
5. Ya estaba en Debian testing/unstable y en Fedora Rawhide y 40 beta, pero en ninguna versión estable. Lasse retiró la puerta trasera y el mantenedor que la agregó desapareció. La identidad detrás de "Jia Tan" no está confirmada públicamente.

La lección: no hackearon el código, hackearon la confianza. Lo que todos revisaban (git) no era lo que se compilaba (el tarball), y los binarios de prueba se trataban como datos.

Fuentes oficiales:
- [backdoor in upstream xz/liblzma leading to ssh server compromise](https://www.openwall.com/lists/oss-security/2024/03/29/4), Andres Freund, oss-security, 2024-03-29.
- [Correos de xz-devel (2021–2022)](https://www.mail-archive.com/xz-devel@tukaani.org/) y [commits del repositorio de xz](https://github.com/tukaani-project/xz).
- [CVE-2024-3094](https://nvd.nist.gov/vuln/detail/CVE-2024-3094) (NVD) y [alerta de Red Hat](https://www.redhat.com/en/blog/urgent-security-alert-fedora-41-and-rawhide-users).

## La demo: `release-diff`
Compara el tarball de una versión con su tag de git **antes de compilarlo** y marca tres cosas:

1. **Archivos que solo existen en el tarball** (los generados típicos de autotools, como `Makefile.in`, se avisan aparte y no bloquean).
2. **Scripts de build que cambian** entre git y tarball (`configure`, `*.m4`, `CMakeLists.txt`, `*.sh`…).
3. **Binarios opacos en los tests**: archivos binarios bajo `tests/` que ningún script del proyecto genera. En xz también estaban en git, así que se marcan aunque el tarball coincida.

```bash
go run github.com/DesiAPG/tech-autopsy/autopsia/L02-xz-2024/cmd/release-diff@latest -demo
# con un proyecto real:
release-diff -tarball proyecto-1.2.tar.gz -tag v1.2 -repo ./proyecto
```

```
$ release-diff --tarball proyecto-1.2.tar.gz --tag v1.2
solo en el tarball: m4/build-to-host.m4   ✗
script de build cambiado: configure       ✗
binario opaco en tests: tests/files/*.xz  ✗
BLOQUEADO: 3 diferencias sin explicar
```

El modo `-demo` usa un proyecto de juguete con el mismo patrón que xz 5.6.0, sin nada malicioso. Sale con código 1 si encuentra algo, para usarlo en CI. Solo librería estándar (más `git` para leer el tag).

Y lo que no es código: mira quién mantiene lo que usas. Si una pieza crítica depende de una sola persona sin apoyo, el riesgo es humano, y apoyarla también es seguridad.
