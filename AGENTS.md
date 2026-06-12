# admiralctl

`admiralctl` es la CLI oficial de Admiral.

Hace:

- inicialización y diagnóstico.
- gestión de nodos, apps, instancias, backups y operaciones.
- interacción con `admirald`.
- salida legible y JSON.

No hace:

- escribir directo en base de datos.
- hablar con workers saltándose `admirald`.
- duplicar lógica de negocio.

Regla práctica:

- la CLI debe exponer el workflow real del producto sin inventar capacidades.
