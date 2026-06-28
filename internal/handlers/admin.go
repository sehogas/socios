package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sehogas/socios/db/sqlc"
	"github.com/sehogas/socios/internal/auth"
	"github.com/sehogas/socios/internal/backup"
	"github.com/sehogas/socios/internal/middleware"
)

type AdminHandler struct {
	queries *sqlc.Queries
	db      *sql.DB
}

func NewAdminHandler(queries *sqlc.Queries, db *sql.DB) *AdminHandler {
	return &AdminHandler{queries: queries, db: db}
}

type DashboardStats struct {
	TotalIngresos         float64
	TotalEgresos          float64
	BalanceNeto           float64
	SociosActivos         int
	SolicitudesPendientes int
	CajaReciente          []sqlc.TransaccionesCaja
	ArchivosBackup        []string
}

// Dashboard gestiona el punto de entrada principal tras iniciar sesión
func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	session := middleware.GetSession(r.Context())
	if session == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := make(map[string]interface{})

	if session.Rol == "admin" {
		// Estadísticas para Administrador
		ingresos, _ := h.queries.GetTotalIngresosCaja(r.Context())
		egresos, _ := h.queries.GetTotalEgresosCaja(r.Context())
		
		// Listar socios para contar estados
		socios, _ := h.queries.ListSocios(r.Context())
		activos := 0
		pendientes := 0
		for _, s := range socios {
			if s.Estado == "Aprobado" && s.Activo == 1 {
				activos++
			} else if s.Estado == "Pendiente" {
				pendientes++
			}
		}

		// Movimientos recientes (hasta 10)
		movimientos, _ := h.queries.ListTransaccionesCaja(r.Context())
		if len(movimientos) > 10 {
			movimientos = movimientos[:10]
		}

		// Leer backups locales
		var backupsList []string
		files, err := os.ReadDir("./backups")
		if err == nil {
			for _, file := range files {
				if !file.IsDir() && filepath.Ext(file.Name()) == ".db" {
					backupsList = append(backupsList, file.Name())
				}
			}
		}

		data["Stats"] = DashboardStats{
			TotalIngresos:         ingresos,
			TotalEgresos:          egresos,
			BalanceNeto:           ingresos - egresos,
			SociosActivos:         activos,
			SolicitudesPendientes: pendientes,
			CajaReciente:          movimientos,
			ArchivosBackup:        backupsList,
		}

	} else {
		// Cargar datos del Socio asociado (por email)
		socio, err := h.queries.GetSocioByEmail(r.Context(), sql.NullString{String: session.Email, Valid: true})
		if err == nil {
			data["Socio"] = socio
			
			// Historial y saldos
			ctaCte, _ := h.queries.ListTransaccionesCtaCteBySocio(r.Context(), socio.ID)
			saldo, _ := h.queries.GetSaldoSocio(r.Context(), socio.ID)
			cuotas, _ := h.queries.ListCuotasGeneradasBySocio(r.Context(), socio.ID)

			data["CtaCte"] = ctaCte
			data["Saldo"] = saldo
			data["Cuotas"] = cuotas
		} else {
			// No vinculado
			data["Socio"] = nil
		}
	}

	RenderTemplate(w, r, "dashboard.html", data)
}

// ListUsers muestra la pantalla de gestión de usuarios del sistema
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	usuarios, err := h.queries.ListUsers(r.Context())
	if err != nil {
		http.Error(w, "Error al obtener usuarios: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Usuarios": usuarios,
	}
	RenderTemplate(w, r, "admin/usuarios.html", data)
}

// ChangeUserRole cambia el rol de un usuario (admin <-> user)
func (h *AdminHandler) ChangeUserRole(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/usuarios?error=ID invalido", http.StatusSeeOther)
		return
	}

	rol := r.FormValue("rol")
	if rol != "admin" && rol != "user" {
		http.Redirect(w, r, "/admin/usuarios?error=Rol invalido", http.StatusSeeOther)
		return
	}

	// Impedir auto-cambiarse el rol para no quedar bloqueado
	session := middleware.GetSession(r.Context())
	if session != nil && session.UserID == id {
		http.Redirect(w, r, "/admin/usuarios?error=No puedes quitarte el rol de Administrador a ti mismo", http.StatusSeeOther)
		return
	}

	err = h.queries.UpdateUserRole(r.Context(), sqlc.UpdateUserRoleParams{
		Rol: rol,
		ID:  id,
	})
	if err != nil {
		http.Redirect(w, r, "/admin/usuarios?error=Error al actualizar rol", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/usuarios?success=Rol actualizado correctamente", http.StatusSeeOther)
}

// ToggleUserActive activa o bloquea una cuenta de usuario
func (h *AdminHandler) ToggleUserActive(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/usuarios?error=ID invalido", http.StatusSeeOther)
		return
	}

	activoStr := r.FormValue("activo")
	activo, err := strconv.ParseInt(activoStr, 10, 64)
	if err != nil {
		activo = 1
	}

	// Impedir auto-bloquearse
	session := middleware.GetSession(r.Context())
	if session != nil && session.UserID == id {
		http.Redirect(w, r, "/admin/usuarios?error=No puedes desactivar tu propia cuenta", http.StatusSeeOther)
		return
	}

	err = h.queries.UpdateUserStatus(r.Context(), sqlc.UpdateUserStatusParams{
		Activo: sql.NullInt64{Int64: activo, Valid: true},
		ID:     id,
	})
	if err != nil {
		http.Redirect(w, r, "/admin/usuarios?error=Error al cambiar estado de usuario", http.StatusSeeOther)
		return
	}

	msg := "Usuario habilitado"
	if activo == 0 {
		msg = "Usuario bloqueado"
	}
	http.Redirect(w, r, "/admin/usuarios?success="+msg, http.StatusSeeOther)
}

// CreateWebUser crea directamente un usuario web (socio o admin) desde administración sin confirmación de email
func (h *AdminHandler) CreateWebUser(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")
	rol := r.FormValue("rol")

	if email == "" || password == "" || (rol != "admin" && rol != "user") {
		http.Redirect(w, r, "/admin/usuarios?error=Completa todos los campos correctamente", http.StatusSeeOther)
		return
	}

	if len(password) < 6 {
		http.Redirect(w, r, "/admin/usuarios?error=La contrasena debe tener al menos 6 caracteres", http.StatusSeeOther)
		return
	}

	_, err := h.queries.GetUserByEmail(r.Context(), email)
	if err == nil {
		http.Redirect(w, r, "/admin/usuarios?error=El correo ya se encuentra registrado", http.StatusSeeOther)
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		http.Redirect(w, r, "/admin/usuarios?error=Error al procesar contrasena", http.StatusSeeOther)
		return
	}

	_, err = h.queries.CreateUser(r.Context(), sqlc.CreateUserParams{
		Email:        email,
		PasswordHash: hash,
		Rol:          rol,
		Activo:       sql.NullInt64{Int64: 1, Valid: true},
		Verificado:   1, // Creado por admin se marca como verificado automáticamente
	})
	if err != nil {
		http.Redirect(w, r, "/admin/usuarios?error=Error al crear usuario", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/usuarios?success=Usuario creado correctamente", http.StatusSeeOther)
}


// TriggerBackup genera una copia física del archivo de base de datos actual
func (h *AdminHandler) TriggerBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
		return
	}

	path, err := backup.CreateBackup(r.Context(), h.db)
	if err != nil {
		http.Redirect(w, r, "/dashboard?error=Error al realizar el backup: "+err.Error(), http.StatusSeeOther)
		return
	}

	filename := filepath.Base(path)
	// Redirigir a descargar directamente y mandar mensaje de exito
	http.Redirect(w, r, fmt.Sprintf("/dashboard?success=Copia de seguridad '%s' creada y guardada en el servidor.&download=%s", filename, filename), http.StatusSeeOther)
}

// DownloadBackup sirve para que el administrador descargue un archivo de backup .db
func (h *AdminHandler) DownloadBackup(w http.ResponseWriter, r *http.Request) {
	archivo := r.URL.Query().Get("archivo")
	if archivo == "" {
		http.Error(w, "Archivo no especificado", http.StatusBadRequest)
		return
	}

	// Sanitizar para evitar Directory Traversal
	filename := filepath.Base(archivo)
	backupPath := filepath.Join("./backups", filename)

	// Verificar si el archivo realmente existe
	info, err := os.Stat(backupPath)
	if os.IsNotExist(err) || info.IsDir() {
		http.Error(w, "Archivo de copia de seguridad no encontrado", http.StatusNotFound)
		return
	}

	// Servir archivo
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	http.ServeFile(w, r, backupPath)
}

// UpdateSystemName actualiza el nombre del sistema en la base de datos
func (h *AdminHandler) UpdateSystemName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
		return
	}

	nombre := strings.TrimSpace(r.FormValue("nombre_sistema"))
	if nombre == "" {
		http.Redirect(w, r, "/dashboard?error=El nombre del sistema no puede estar vacio", http.StatusSeeOther)
		return
	}

	err := h.queries.SetConfig(r.Context(), sqlc.SetConfigParams{
		Clave: "nombre_sistema",
		Valor: nombre,
	})
	if err != nil {
		http.Redirect(w, r, "/dashboard?error=Error al guardar configuracion: "+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/dashboard?success=Nombre del sistema actualizado correctamente", http.StatusSeeOther)
}
