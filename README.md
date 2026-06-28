# Socios - Sistema de Socios y Gestión Contable

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://golang.org)
[![Go Reference](https://pkg.go.dev/badge/github.com/sehogas/socios.svg)](https://pkg.go.dev/github.com/sehogas/socios)
[![Go Report Card](https://goreportcard.com/badge/github.com/sehogas/socios)](https://goreportcard.com/report/github.com/sehogas/socios)
[![Tests](https://img.shields.io/badge/tests-passed-brightgreen?style=flat-square&logo=go)](file:///home/shogas/go/src/github.com/sehogas/socios)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square)](file:///home/shogas/go/src/github.com/sehogas/socios/LICENSE)
[![Database: SQLite](https://img.shields.io/badge/Database-SQLite-003B57?style=flat-square&logo=sqlite&logoColor=white)](https://sqlite.org)
[![CGO Free](https://img.shields.io/badge/CGO--Free-modernc--sqlite-brightgreen?style=flat-square)](https://modernc.org/sqlite)
[![sqlc](https://img.shields.io/badge/Generated%20by-sqlc-blue?style=flat-square)](https://sqlc.dev)
[![SweetAlert2](https://img.shields.io/badge/UI-SweetAlert2-E89088?style=flat-square)](https://sweetalert2.github.io)
[![GitHub release](https://img.shields.io/github/v/release/sehogas/socios?style=flat-square)](https://github.com/sehogas/socios/releases)
[![GitHub stars](https://img.shields.io/github/stars/sehogas/socios?style=flat-square&color=blue)](https://github.com/sehogas/socios/stargazers)


Socios es una plataforma monolítica diseñada específicamente para la administración de fichas de socios, cuentas corrientes individuales, tesorería general (caja), un mini-CMS de contenidos y copias de seguridad consistentes en caliente.

El sistema fue concebido para digitalizar el flujo de inscripción física (basado en la planilla del Centro de Residentes Formoseños en Tierra del Fuego), automatizar la generación y cobro de cuotas mensuales, y facilitar un balance de ingresos y egresos a través de un dashboard.

---

## 1. Stack Tecnológico e Infraestructura

*   **Lenguaje y Backend:** Go 1.22+ (utilizando únicamente la biblioteca estándar `net/http` para ruteo).
*   **Base de Datos:** SQLite.
*   **Driver SQLite:** `modernc.org/sqlite` (implementación pura de Go, **libre de CGO**). Esto permite compilar el proyecto de forma estática en cualquier entorno (incluyendo Docker y compilación cruzada) sin necesidad de disponer de un compilador de C (`gcc`).
*   **Acceso a Datos:** **sqlc** (generación de código Go a partir de consultas SQL estrictamente tipadas).
*   **Interfaz de Usuario (UI):** Renderizado en el servidor mediante el motor nativo `html/template`. Estilos en CSS puro (Vanilla CSS) estructurados de forma responsiva (*mobile-first*) con Grid y Flexbox. Interactividad básica en Vanilla JS.
*   **Empaquetado (Assets):** Todos los archivos de plantillas HTML, CSS y JS se compilan dentro del binario final utilizando el paquete nativo `embed` de Go. El programa es auto-contenido y ejecutable en un solo archivo.

---

## 2. Arquitectura de Directorios

El código sigue una estructura limpia y desacoplada típica en el ecosistema Go:

```text
socios/
├── cmd/
│   └── server/
│       └── main.go           # Punto de entrada. Inicializa DB, ejecuta migraciones y levanta el servidor HTTP.
├── db/
│   ├── db.go                 # Expone el esquema SQL embebido para las migraciones automáticas.
│   ├── schema.sql            # Definición DDL (tablas y restricciones).
│   ├── query.sql             # Consultas SQL anotadas para sqlc.
│   └── sqlc/                 # Código Go tipado generado automáticamente por sqlc.
├── internal/
│   ├── auth/
│   │   ├── hash.go           # Utilidades de hashing de contraseñas usando bcrypt.
│   │   └── session.go        # Cifrado y descifrado de sesiones en cookies mediante AES-GCM.
│   ├── backup/
│   │   └── backup.go         # Lógica de respaldo seguro y caliente usando VACUUM INTO.
│   ├── database/
│   │   ├── db.go             # Inicialización y configuración de conexión SQLite (modo WAL y FKs).
│   │   └── migrations.go     # Verificación e inicialización automática del esquema de base de datos.
│   ├── middleware/
│   │   └── auth.go           # Middlewares de control de acceso, verificación de roles y de socio activo.
│   ├── handlers/
│   │   ├── admin.go          # Dashboard principal, gestión de usuarios del sistema y backups.
│   │   ├── auth.go           # Formulario y endpoints de login, registro y cierre de sesión.
│   │   ├── cms.go            # Visualización pública y administración del CMS.
│   │   ├── cuentas.go        # Gestión de movimientos de caja y facturación masiva de cuotas.
│   │   ├── socios.go         # CRUD de socios, cobro/adeudo de cuenta corriente y pagos de cuotas.
│   │   └── render.go         # Helper para inyectar datos globales y procesar templates HTML embebidos.
│   └── server/
│       └── server.go         # Definición del multiplexer HTTP y enrutamiento con middlewares.
├── web/
│   ├── embed.go              # Directiva go:embed para compilar las carpetas templates y static.
│   ├── static/
│   │   ├── css/style.css     # Hoja de estilos responsiva y minimalista.
│   │   └── js/main.js        # Script de confirmaciones y alertas en el cliente.
│   └── templates/            # Plantillas Go HTML organizadas por sección.
├── Dockerfile                # Compilación multi-stage y empaquetado del contenedor.
├── go.mod
└── sqlc.yaml                 # Configuración de sqlc.
```

---

## 3. Lógica Contable y Flujos del Negocio

### A. Vinculación Socio - Usuario Web
*   Los usuarios se registran libremente en la web usando un correo electrónico obligatorio. Al registrarse son usuarios huérfanos independientes de la parte contable del club.
*   El administrador carga la ficha digitalizada a partir de la planilla física de inscripción.
*   Cuando el administrador asigna e ingresa el mismo correo electrónico en la ficha de socio que el usuario usó en su registro, el sistema los asocia automáticamente.
*   Una vez asociados, el socio puede visualizar su estado de cuenta corriente, sus cuotas pendientes y su historial de pagos desde su panel personal.

### B. Relación Cuenta Corriente vs. Caja General
El sistema divide los movimientos en dos universos que se comunican:
1.  **Cuenta Corriente del Socio:** Mide la deuda individual.
    *   *Débitos:* Cargos de cuota mensual o alquileres. Aumentan la deuda del socio.
    *   *Créditos:* Pagos realizados por el socio o dinero entregado como adelanto. Reducen la deuda o crean saldo a favor.
2.  **Caja General del Club (Tesorería):** Registra el flujo de fondos real (ingresos/egresos) distribuidos en cuentas de efectivo, banco o MercadoPago.

**Interconexión en los Cobros:**
*   Cuando un administrador registra un **Crédito** en la Cuenta Corriente de un socio, puede seleccionar a qué cuenta real del club ingresó el dinero (ej. Caja Efectivo, Banco).
*   Al guardar el movimiento, una transacción de base de datos segura inserta el crédito en la cuenta corriente del socio y crea de forma atómica un registro de **Ingreso** en la Caja General del club.
*   Si se registra una corrección de saldo que no involucra efectivo, se puede seleccionar "No registrar en Caja".

### C. Generación y Pago de Cuotas
*   **Generación Mensual:** El administrador puede facturar de forma masiva a todos los socios aprobados y activos. Esto inserta un registro en la tabla `cuotas_generadas` (estado `Impaga`) y genera un **Débito** en la Cuenta Corriente de cada socio por el valor configurado.
*   **Uso de Adelantos (Pago desde Cta Cte):** Si un socio tiene saldo a favor en su Cuenta Corriente (registrado previamente como un crédito/adelanto), el administrador puede seleccionar la cuota impaga y presionar "Pagar desde Cta Cte". Esto:
    1.  Verifica que el saldo a favor sea suficiente.
    2.  Registra un **Débito** en la Cuenta Corriente por el cobro de la cuota.
    3.  Actualiza el estado de la cuota generada a `Paga`.

---

## 4. Diseño de Seguridad

*   **Contraseñas:** Encriptadas en base de datos usando **bcrypt** con costo por defecto.
*   **Sesiones (Client-Side Encrypted Cookies):** En lugar de persistir sesiones en la base de datos y realizar lecturas por cada recurso solicitado, Socios cifra la estructura de sesión en formato JSON utilizando **AES-GCM (criptografía autenticada)**.
    *   La clave de cifrado de 32 bytes se genera de forma aleatoria en el arranque del servidor y se guarda localmente en el archivo `.session_key`. Esto permite mantener las sesiones activas de los usuarios incluso si el servidor se reinicia.
    *   Las cookies se configuran como `HTTP-Only`, `SameSite=Lax` y con banderas de seguridad correspondientes.
*   **Middleware de Socio Activo:** Al intentar acceder a secciones restringidas de socios, el middleware valida que la cuenta del socio vinculada al email tenga la propiedad `activo = 1` y `estado = 'Aprobado'`. Si la administración inhabilita a un socio (por morosidad u otra causa), su acceso web a contenido restringido se bloquea de forma inmediata.

---

## 5. Instrucciones para el Desarrollador

### Requisitos
*   Go (v1.22+) instalado localmente.
*   Binario de `sqlc` instalado (opcional, necesario si cambias las consultas SQL):
    ```bash
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
    ```

### Ejecución en Modo Desarrollo
Para iniciar el servidor localmente con recarga manual:
```bash
# Descargar dependencias declaradas en go.mod
go mod tidy

# Iniciar servidor local
go run cmd/server/main.go
```
El servidor escuchará por defecto en `http://localhost:8080`.

*Nota de Inicialización:* La base de datos se inicializa automáticamente al detectar que está vacía. El primer usuario que se registre a través del formulario web (`/register`) será promovido de manera automática al rol de **Administrador**.

### Modificaciones en Base de Datos (Flujo de Trabajo con sqlc)
Si deseas agregar columnas o nuevas tablas:
1.  Modifica el archivo DDL de base de datos en [db/schema.sql](file:///home/shogas/go/src/github.com/sehogas/socios/db/schema.sql).
2.  Escribe las consultas correspondientes en [db/query.sql](file:///home/shogas/go/src/github.com/sehogas/socios/db/query.sql) utilizando las anotaciones de sqlc.
3.  Regenera el código Go corriendo el compilador de sqlc en la raíz del proyecto:
    ```bash
    sqlc generate
    ```
4.  Implementa la lógica del controlador correspondiente en `internal/handlers/` consumiendo los métodos autogenerados.

---

## 6. Despliegue y Ciclo de Vida

El proyecto incluye configuraciones listas para usar tanto en desarrollo como en producción mediante Docker y Docker Compose, además de entrega continua (CD) mediante GitHub Actions.

### A. Despliegue con Docker Compose

#### Desarrollo
Para arrancar el entorno de desarrollo local con Docker Compose (construyendo la imagen localmente y usando volúmenes de host para fácil depuración):
```bash
docker-compose up -d --build
```
*   El servidor estará disponible en `http://localhost:8080`.
*   La base de datos SQLite se mantendrá persistente en el directorio `./data/` de tu máquina.
*   Las copias de seguridad de la base de datos se guardarán localmente en `./backups/`.

#### Producción
Para desplegar en producción utilizando la imagen empaquetada oficial en GitHub Packages (GHCR) y volúmenes lógicos con nombre administrados por Docker (práctica recomendada):
```bash
docker-compose -f docker-compose-prod.yml up -d
```
*   El servidor estará expuesto en el puerto estándar `80` del host (mapeado al puerto interno `8080`).
*   La persistencia de datos y las copias de seguridad se almacenan en los volúmenes aislados `socios_data` y `socios_backups`.

---

### B. Endpoint de Salud y Versión

El servidor expone un endpoint público para verificar el estado del servicio y la versión del binario en ejecución:

*   **Ruta:** `GET /health`
*   **Respuesta Exitosa (HTTP 200):**
    ```json
    {
      "status": "ok",
      "version": "v1.0.0"
    }
    ```
*   **Respuesta de Error (HTTP 500):** Se emite en caso de que la base de datos no sea accesible:
    ```json
    {
      "status": "error",
      "version": "v1.0.0"
    }
    ```

---

### C. Automatización y CI/CD en GitHub (GitHub Actions)

El proyecto cuenta con un workflow configurado en [.github/workflows/release.yml](file:///.github/workflows/release.yml) para automatizar la publicación de nuevas versiones y el empaquetado multiplataforma.

#### Flujo de Publicación:
1. Crea un nuevo tag o etiqueta de versión que comience con la letra `v` (por ejemplo, `v1.0.0`):
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```
2. El workflow de GitHub Actions se activará de forma automática ejecutando:
   *   **Compilación Cruzada de Binarios Estáticos**: Compila binarios independientes sin CGO para las siguientes arquitecturas:
       *   **Linux** (AMD64 y ARM64)
       *   **Windows** (AMD64 y ARM64)
       *   **macOS / Mac** (AMD64 y ARM64)
   *   **Construcción e inyección en Docker**: Compila y publica la imagen oficial en el registro de GitHub Packages (GHCR) (`ghcr.io/sehogas/socios:latest` y `ghcr.io/sehogas/socios:v1.0.0`) inyectando el número de la versión dentro de la imagen.
   *   **GitHub Release**: Crea una publicación de versión en la sección "Releases" de GitHub cargando los 6 binarios multiplataforma listos para su descarga.

---

### D. Copias de Seguridad (Backups)
*   **Generación:** En el panel de control del administrador, al pulsar "Generar Backup", el sistema ejecuta un respaldo en caliente. Este archivo se escribe físicamente en `/app/backups/backup_YYYYMMDD_HHMMSS.db`.
*   **Descarga:** Al generarse, el navegador web iniciará automáticamente la descarga del archivo `.db` en tu computadora.
*   **Restauración:** Para restaurar el sistema a un punto anterior:
    1.  Detén el contenedor o servidor.
    2.  Reemplaza el archivo activo `database.db` (ubicado en tu volumen `data/`) con una copia del backup renombrada a `database.db`.
    3.  Vuelve a iniciar el servidor.

### E. Variables de Entorno de Configuración

Puedes modificar el comportamiento inicial del binario o del contenedor utilizando las siguientes variables de entorno:

| Variable | Descripción | Valor por Defecto |
| --- | --- | --- |
| `PORT` | Puerto HTTP donde escuchará la aplicación. | `8080` |
| `DATABASE_PATH` | Ruta al archivo de base de datos SQLite (ej. `./database.db` o `/app/data/database.db` en Docker). | `./database.db` |
| `ENV` | Perfil del entorno: `development` (desarrollo) o `production` (producción). | `development` |
| `SESSION_SECRET` | Clave de cifrado de 32 bytes para las cookies de sesión (AES-GCM). En caso de no proveerse, se genera una clave aleatoria en el arranque guardada en el archivo local `.session_key`. Se recomienda configurar una clave fija en producción. | *(Autogenerada)* |
| `SMTP_HOST` | Host o dirección del servidor SMTP para el envío de correos. Si se deja en blanco, el envío de correos se simulará en la consola del servidor (modo desarrollo). | *(Vacío / Simulado)* |
| `SMTP_PORT` | Puerto del servidor de correo SMTP. | `587` |
| `SMTP_USER` | Usuario de autenticación para el servidor SMTP. | *(Vacío)* |
| `SMTP_PASS` | Contraseña de autenticación para el servidor SMTP. | *(Vacío)* |
| `SMTP_FROM` | Dirección de correo del remitente para las notificaciones enviadas. | `no-reply@tu-club.com` |
