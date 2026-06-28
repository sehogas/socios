# Guía de Configuración Inicial y Primeros Pasos 🚀

Bienvenido al sistema **Club de Socios**. Esta guía detalla la configuración inicial y las acciones necesarias para poner en marcha la plataforma.

---

## 🛠️ 1. Configuración del Entorno (`.env`)

El sistema lee las configuraciones desde variables de entorno. Puedes basarte en el archivo `.env.template` en la raíz del proyecto para crear tu archivo `.env`.

### Variables clave a configurar:
| Variable | Descripción | Valor por Defecto / Sugerido |
| :--- | :--- | :--- |
| `PORT` | Puerto de escucha del servidor web. | `8080` |
| `DB_PATH` | Ruta al archivo de base de datos SQLite. | `./data/database.db` |
| `SMTP_HOST` | Host SMTP para enviar correos de verificación. | *(Vacío = Modo Consola/Simulado)* |
| `SMTP_PORT` | Puerto del servidor SMTP. | `587` |
| `SMTP_USER` | Usuario/Correo de autenticación SMTP. | *(Tu usuario de correo)* |
| `SMTP_PASS` | Contraseña del servidor SMTP. | *(Tu contraseña de aplicación)* |
| `SMTP_FROM` | Remitente de los correos automáticos. | `no-reply@tu-club.com` |

> [!NOTE]
> **Modo Simulación SMTP:** Si dejas `SMTP_HOST` en blanco, el sistema simulará el envío de correos imprimiendo los enlaces de confirmación y recuperación directamente en la consola del servidor. Esto es ideal para entornos de desarrollo y pruebas rápidas.

---

## 💾 2. Inicialización de la Base de Datos

El sistema utiliza **SQLite** y autogestiona su esquema.
* Las migraciones y la creación de tablas se ejecutan de manera automática al levantar el servidor por primera vez gracias a la función `RunMigrations`.
* No es necesario ejecutar scripts SQL manuales.

---

## 🔑 3. Creación del Primer Usuario Administrador

El sistema implementa una lógica de auto-aprovisionamiento seguro para el primer usuario:

1. Levanta el servidor ejecutando `make run` o mediante el contenedor Docker.
2. Abre tu navegador en `http://localhost:8080/register`.
3. Completa el formulario de registro con tus datos y haz clic en **Registrarse**.

> [!IMPORTANT]
> **Promoción Automática a Administrador:**
> El **primer** usuario en registrarse en la base de datos se convertirá **automáticamente** en **Administrador** y su cuenta quedará **Verificada** al instante, saltándose la validación por correo electrónico.

---

## ⚙️ 4. Configuración Básica del Club

Una vez dentro como Administrador, realiza estos primeros pasos para personalizar la plataforma:

### A. Nombre del Club
1. Ve al **Inicio** (Dashboard).
2. En la sección **Configuración General** (abajo a la izquierda), cambia el **Nombre del Sistema** por el nombre real de tu club.
3. Haz clic en **Guardar Nombre**. Este nombre se reflejará en la barra de navegación y los correos electrónicos.

### B. Valores de Cuota Social
Es **obligatorio** configurar al menos las clasificaciones de cuota que usarán tus socios iniciales, ya que el sistema bloqueará la generación de cuotas mensuales si detecta socios activos con clasificaciones sin tarifa configurada.
1. Ve a **Cuentas** en la barra de navegación.
2. Desplázate a **Gestión de Valores de Cuota Social**.
3. Carga un valor de monto y vigencia inicial (formato AAAA-MM) para las clasificaciones necesarias (ej. `Titular`, `Adherente`).
4. Guarda las configuraciones.
