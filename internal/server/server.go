package server

import (
	"database/sql"
	"io/fs"
	"net/http"

	"github.com/sehogas/socios3/db/sqlc"
	"github.com/sehogas/socios3/internal/handlers"
	"github.com/sehogas/socios3/internal/middleware"
	"github.com/sehogas/socios3/web"
)

// NewServer inicializa todos los manejadores, configura el ruteo
// con middlewares y devuelve un http.Handler listo para escuchar peticiones.
func NewServer(db *sql.DB) http.Handler {
	queries := sqlc.New(db)
	handlers.SetDatabase(db, queries)
	mw := middleware.NewMiddleware(queries)

	mux := http.NewServeMux()

	// Inicializar Handlers
	authH := handlers.NewAuthHandler(queries, db)
	sociosH := handlers.NewSociosHandler(queries, db)
	cuentasH := handlers.NewCuentasHandler(queries, db)
	adminH := handlers.NewAdminHandler(queries, db)
	cmsH := handlers.NewCMSHandler(queries)

	// Servir archivos estáticos embebidos
	staticFS, err := fs.Sub(web.Assets, "static")
	if err != nil {
		panic("error al leer archivos estaticos embebidos: " + err.Error())
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Redirección raíz
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		// Redirigir al dashboard si está autenticado, sino a ver CMS público
		session := middleware.GetSession(r.Context())
		if session != nil {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/cms/ver", http.StatusSeeOther)
		}
	})

	// Rutas Públicas (Autenticación y CMS)
	mux.HandleFunc("GET /login", authH.ShowLogin)
	mux.HandleFunc("POST /login", authH.Login)
	mux.HandleFunc("GET /register", authH.ShowRegister)
	mux.HandleFunc("POST /register", authH.Register)
	mux.HandleFunc("GET /logout", authH.Logout)
	mux.HandleFunc("GET /cms/ver", cmsH.ViewCMS)

	// Nuevas Rutas de Verificación y Recuperación de Contraseña (Públicas)
	mux.HandleFunc("GET /verificar-email", authH.VerifyEmail)
	mux.HandleFunc("GET /recover-password", authH.ShowRecoverPassword)
	mux.HandleFunc("POST /recover-password", authH.RecoverPassword)
	mux.HandleFunc("GET /restablecer-password", authH.ShowResetPassword)
	mux.HandleFunc("POST /restablecer-password", authH.ResetPassword)

	// Middleware de Administración (Requiere Auth + Rol admin)
	adminOnly := func(h http.HandlerFunc) http.Handler {
		return mw.RequireAuth(mw.RequireRole("admin")(h))
	}

	// Rutas Privadas Comunes (Requiere Auth)
	mux.Handle("GET /dashboard", mw.RequireAuth(http.HandlerFunc(adminH.Dashboard)))
	mux.Handle("POST /cambiar-password", mw.RequireAuth(http.HandlerFunc(authH.ChangePassword)))

	// Rutas de Administración de Socios
	mux.Handle("GET /admin/socios", adminOnly(sociosH.ListSocios))
	mux.Handle("GET /admin/socios/nuevo", adminOnly(sociosH.ShowNewSocioForm))
	mux.Handle("POST /admin/socios/nuevo", adminOnly(sociosH.CreateSocio))
	mux.Handle("GET /admin/socios/editar", adminOnly(sociosH.ShowEditSocioForm))
	mux.Handle("POST /admin/socios/editar", adminOnly(sociosH.UpdateSocio))
	mux.Handle("GET /admin/socios/detalle", adminOnly(sociosH.SocioDetail))
	mux.Handle("POST /admin/socios/cta-cte/nuevo", adminOnly(sociosH.CreateCtaCteTransaction))
	mux.Handle("POST /admin/socios/cuotas/pagar", adminOnly(sociosH.PayQuotaFromCtaCte))
	mux.Handle("GET /admin/socios/aprobar", adminOnly(sociosH.ApproveSocio))
	mux.Handle("GET /admin/socios/rechazar", adminOnly(sociosH.RejectSocio))
	mux.Handle("GET /admin/socios/toggle-activo", adminOnly(sociosH.ToggleActivo))

	// Rutas de Administración de Caja y Cuentas
	mux.Handle("GET /admin/cuentas", adminOnly(cuentasH.CuentasGeneral))
	mux.Handle("POST /admin/cuentas/nueva-transaccion", adminOnly(cuentasH.CreateCajaTransaction))
	mux.Handle("POST /admin/cuotas/generar", adminOnly(cuentasH.GenerateMonthlyQuotas))
	mux.Handle("POST /admin/cuotas/valores", adminOnly(cuentasH.UpdateCuotaValor))

	// Rutas de Administración de Usuarios del Sistema
	mux.Handle("GET /admin/usuarios", adminOnly(adminH.ListUsers))
	mux.Handle("POST /admin/usuarios/cambiar-rol", adminOnly(adminH.ChangeUserRole))
	mux.Handle("POST /admin/usuarios/toggle-activo", adminOnly(adminH.ToggleUserActive))
	mux.Handle("POST /admin/usuarios/nuevo", adminOnly(adminH.CreateWebUser)) // Nueva ruta genérica de creación sin mail confirm

	// Rutas de Administración de Contenidos (CMS)
	mux.Handle("GET /admin/paginas", adminOnly(cmsH.AdminCMS))
	mux.Handle("POST /admin/paginas/nueva", adminOnly(cmsH.CreatePage))
	mux.Handle("POST /admin/paginas/editar", adminOnly(cmsH.UpdatePage))
	mux.Handle("POST /admin/paginas/eliminar", adminOnly(cmsH.DeletePage))

	// Rutas de Copias de Seguridad (Backups)
	mux.Handle("POST /admin/backup/crear", adminOnly(adminH.TriggerBackup))
	mux.Handle("GET /admin/backup/descargar", adminOnly(adminH.DownloadBackup))

	// Configuración del Sistema
	mux.Handle("POST /admin/config/nombre", adminOnly(adminH.UpdateSystemName))

	// Aplicar el cargador de sesión global sobre todas las rutas
	return mw.SessionLoader(mux)
}
