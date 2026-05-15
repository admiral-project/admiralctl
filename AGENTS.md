admiralctl
Definición de producto

admiralctl es la interfaz de línea de comandos oficial de Admiral. Permite a operadores, administradores y desarrolladores interactuar con admirald desde terminal para instalar, configurar, diagnosticar y operar la plataforma.

admiralctl debe ser una herramienta simple, clara y confiable para administrar Admiral sin depender exclusivamente de interfaces web.

Propósito principal

Permitir la operación técnica de Admiral desde consola, facilitando instalación, configuración, diagnóstico, automatización y administración de recursos.

Responsabilidades principales

admiralctl debe permitir:

Inicializar una instalación de Admiral.
Configurar modo single node.
Registrar nodos.
Consultar estado de la plataforma.
Gestionar definiciones de apps.
Gestionar tiers.
Consultar apps contratadas.
Ejecutar operaciones administrativas.
Consultar logs y eventos relevantes.
Ejecutar diagnósticos.
Validar archivos YAML de aplicaciones.
Facilitar troubleshooting.
Automatizar tareas mediante scripts.
Lo que admiralctl sí debe hacer

admiralctl debe:

Comunicarse con admirald.
Leer configuración local.
Autenticarse contra admirald.
Ejecutar comandos administrativos.
Mostrar resultados claros en terminal.
Soportar salida en formato humano y JSON.
Validar archivos antes de enviarlos.
Ser útil en instalaciones single node y distribuidas.
Ayudar a instalar y bootstrapear Admiral.
Lo que admiralctl no debe hacer

admiralctl no debe:

Reemplazar a admirald.
Escribir directamente en la base de datos de Admiral.
Ejecutar directamente tareas en workers salvo operaciones locales muy controladas de diagnóstico.
Contener lógica de negocio crítica que no exista en admirald.
Ser una UI interactiva compleja.
Convertirse en una herramienta pesada o difícil de usar.
Funciones mínimas del MVP
Inicialización
admiralctl init
admiralctl init --mode single-node --domain apps.example.com
admiralctl init --mode worker --server https://admiral.example.com
Estado general
admiralctl status
admiralctl health
admiralctl doctor
Gestión de nodos
admiralctl nodes list
admiralctl nodes show NODE_ID
admiralctl nodes register
admiralctl nodes disable NODE_ID
admiralctl nodes enable NODE_ID
Gestión de apps
admiralctl apps list
admiralctl apps show APP_NAME
admiralctl apps apply -f app.yaml
admiralctl apps validate -f app.yaml
admiralctl apps disable APP_NAME
admiralctl apps enable APP_NAME
Gestión de instancias
admiralctl instances list
admiralctl instances show INSTANCE_ID
admiralctl instances pause INSTANCE_ID
admiralctl instances resume INSTANCE_ID
admiralctl instances resize INSTANCE_ID --tier business
admiralctl instances deprovision INSTANCE_ID
Backups
admiralctl backups list
admiralctl backups create INSTANCE_ID
admiralctl backups show BACKUP_ID
Operaciones
admiralctl operations list
admiralctl operations show OPERATION_ID
admiralctl operations retry OPERATION_ID
Configuración
admiralctl config show
admiralctl config set server https://admiral.example.com
admiralctl login
admiralctl logout
Experiencia esperada

La CLI debe ser predecible y amigable.

Ejemplo:

admiralctl status

Salida:

Admiral status

API:        healthy
Database:   healthy
RabbitMQ:   healthy
Redis:      healthy
Fleet:      3 active nodes
Apps:       18 running
Backups:    0 running

Con salida JSON:

admiralctl status --output json
Requerimientos técnicos
Lenguaje: Go.
Binario único.
Configuración en archivo local.
Soporte para variables de entorno.
Salida en tabla, texto y JSON.
Manejo claro de errores.
Códigos de salida adecuados para scripts.
Compatible con Linux.
Distribuible como RPM.
Documentación integrada por comando.
Autocompletado shell como mejora posterior.
Criterio de éxito

admiralctl será exitoso si un operador puede instalar, diagnosticar y administrar Admiral desde terminal de forma rápida, clara y segura, sin tener que manipular manualmente la base de datos, los servicios internos o los workers.
