package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sehogas/socios3/db/sqlc"
	"github.com/sehogas/socios3/internal/auth"
	"github.com/sehogas/socios3/internal/middleware"
)

type AuthHandler struct {
	queries *sqlc.Queries
	db      *sql.DB
}

func NewAuthHandler(queries *sqlc.Queries, db *sql.DB) *AuthHandler {
	return &AuthHandler{queries: queries, db: db}
}

// ShowLogin muestra el formulario de inicio de sesión
func (h *AuthHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, r, "login.html", nil)
}

// Login maneja el POST de credenciales y establece la cookie de sesión
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	if email == "" || password == "" {
		http.Redirect(w, r, "/login?error=Completa todos los campos", http.StatusSeeOther)
		return
	}

	user, err := h.queries.GetUserByEmail(r.Context(), email)
	if err != nil {
		http.Redirect(w, r, "/login?error=Credenciales incorrectas", http.StatusSeeOther)
		return
	}

	if user.Activo.Valid && user.Activo.Int64 == 0 {
		http.Redirect(w, r, "/login?error=Tu cuenta de usuario ha sido suspendida", http.StatusSeeOther)
		return
	}

	// Requisito: Verificar si el correo ha sido validado
	if user.Verificado == 0 {
		http.Redirect(w, r, "/login?error=Debes verificar tu correo electronico antes de ingresar. Revisa tu casilla de entrada.", http.StatusSeeOther)
		return
	}

	if !auth.CheckPasswordHash(password, user.PasswordHash) {
		http.Redirect(w, r, "/login?error=Credenciales incorrectas", http.StatusSeeOther)
		return
	}

	// Crear sesión
	session := &auth.Session{
		UserID:    user.ID,
		Email:     user.Email,
		Rol:       user.Rol,
		CreatedAt: time.Now(),
	}

	err = auth.SetSessionCookie(w, session)
	if err != nil {
		http.Redirect(w, r, "/login?error=Error al iniciar sesion", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// ShowRegister muestra el formulario de registro de usuario web
func (h *AuthHandler) ShowRegister(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, r, "register.html", nil)
}

// Register maneja la creación de la cuenta web por autoregistro con confirmación SMTP
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	if email == "" || password == "" {
		http.Redirect(w, r, "/register?error=Completa todos los campos", http.StatusSeeOther)
		return
	}

	if len(password) < 6 {
		http.Redirect(w, r, "/register?error=La contrasena debe tener al menos 6 caracteres", http.StatusSeeOther)
		return
	}

	// Comprobar si el email ya existe
	_, err := h.queries.GetUserByEmail(r.Context(), email)
	if err == nil {
		http.Redirect(w, r, "/register?error=El correo ya se encuentra registrado", http.StatusSeeOther)
		return
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		http.Redirect(w, r, "/register?error=Error al procesar contrasena", http.StatusSeeOther)
		return
	}

	// Comprobar si es el primer usuario en la base de datos (se le asigna admin y verificado = 1 directo)
	users, err := h.queries.ListUsers(r.Context())
	rol := "user"
	verificado := int64(0)
	if err != nil || len(users) == 0 {
		rol = "admin"
		verificado = 1
	}

	// Transacción para insertar el usuario y generar el token de verificación si aplica
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Redirect(w, r, "/register?error=Error de conexion", http.StatusSeeOther)
		return
	}
	defer tx.Rollback()

	qtx := h.queries.WithTx(tx)

	// Crear usuario
	dbUser, err := qtx.CreateUser(r.Context(), sqlc.CreateUserParams{
		Email:        email,
		PasswordHash: passwordHash,
		Rol:          rol,
		Activo:       sql.NullInt64{Int64: 1, Valid: true},
		Verificado:   verificado,
	})
	if err != nil {
		http.Redirect(w, r, "/register?error=Error al registrar usuario", http.StatusSeeOther)
		return
	}

	// Si no es el primer usuario, requiere verificación por correo
	if verificado == 0 {
		token := generateSecureToken()
		expiracion := time.Now().Add(24 * time.Hour)

		_, err = qtx.CreateToken(r.Context(), sqlc.CreateTokenParams{
			UsuarioID:  dbUser.ID,
			Token:      token,
			Tipo:       "VERIFICACION_EMAIL",
			Expiracion: expiracion,
		})
		if err != nil {
			http.Redirect(w, r, "/register?error=Error al generar token de verificacion", http.StatusSeeOther)
			return
		}

		// Construir URL base
		baseURL := getBaseURL(r)
		err = auth.SendVerificationEmail(baseURL, email, token)
		if err != nil {
			http.Redirect(w, r, "/register?error=Error al enviar correo de verificacion: "+err.Error(), http.StatusSeeOther)
			return
		}
	}

	tx.Commit()

	if verificado == 1 {
		http.Redirect(w, r, "/login?success=Registro exitoso. Eres el primer administrador, ingresa directamente.", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/login?success=Registro exitoso. Se ha enviado un correo de confirmacion. Revisa tu casilla.", http.StatusSeeOther)
	}
}

// VerifyEmail maneja la validación de cuenta al presionar el enlace de confirmación
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	tokenVal := r.URL.Query().Get("token")
	if tokenVal == "" {
		http.Redirect(w, r, "/login?error=Token ausente", http.StatusSeeOther)
		return
	}

	// Buscar token
	tokenObj, err := h.queries.GetTokenByValueAndTipo(r.Context(), sqlc.GetTokenByValueAndTipoParams{
		Token: tokenVal,
		Tipo:  "VERIFICACION_EMAIL",
	})
	if err != nil {
		http.Redirect(w, r, "/login?error=Enlace de confirmacion invalido", http.StatusSeeOther)
		return
	}

	// Comprobar expiración
	if tokenObj.Expiracion.Before(time.Now()) {
		// Borrar token expirado
		_ = h.queries.DeleteToken(r.Context(), tokenObj.ID)
		http.Redirect(w, r, "/login?error=El enlace de confirmacion ha expirado. Por favor registrate nuevamente.", http.StatusSeeOther)
		return
	}

	// Transacción para marcar verificado y eliminar token
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Redirect(w, r, "/login?error=Error de base de datos", http.StatusSeeOther)
		return
	}
	defer tx.Rollback()

	qtx := h.queries.WithTx(tx)

	// Activar usuario
	err = qtx.UpdateUserVerification(r.Context(), sqlc.UpdateUserVerificationParams{
		Verificado: 1,
		ID:         tokenObj.UsuarioID,
	})
	if err != nil {
		http.Redirect(w, r, "/login?error=Error al verificar la cuenta", http.StatusSeeOther)
		return
	}

	// Eliminar token usado
	err = qtx.DeleteToken(r.Context(), tokenObj.ID)
	if err != nil {
		http.Redirect(w, r, "/login?error=Error al limpiar token", http.StatusSeeOther)
		return
	}

	tx.Commit()
	http.Redirect(w, r, "/login?success=Tu cuenta ha sido verificada con exito. Ya puedes iniciar sesion.", http.StatusSeeOther)
}

// ShowRecoverPassword muestra la pantalla para ingresar correo de recuperación
func (h *AuthHandler) ShowRecoverPassword(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, r, "recover_password.html", nil)
}

// RecoverPassword maneja el pedido de recuperación enviando un correo SMTP
func (h *AuthHandler) RecoverPassword(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	if email == "" {
		http.Redirect(w, r, "/recover-password?error=Ingresa un correo valido", http.StatusSeeOther)
		return
	}

	// Buscar si el usuario existe
	user, err := h.queries.GetUserByEmail(r.Context(), email)
	if err != nil {
		// Mensaje genérico para evitar enumeración de cuentas
		http.Redirect(w, r, "/login?success=Si el correo se encuentra registrado, recibiras las instrucciones de recuperacion en instantes.", http.StatusSeeOther)
		return
	}

	// Transacción para limpiar tokens viejos y guardar el nuevo
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Redirect(w, r, "/recover-password?error=Error interno de conexion", http.StatusSeeOther)
		return
	}
	defer tx.Rollback()

	qtx := h.queries.WithTx(tx)

	// Limpiar tokens anteriores de password
	_ = qtx.DeleteTokensByUsuarioAndTipo(r.Context(), sqlc.DeleteTokensByUsuarioAndTipoParams{
		UsuarioID: user.ID,
		Tipo:      "RECUPERACION_PASSWORD",
	})

	token := generateSecureToken()
	expiracion := time.Now().Add(1 * time.Hour) // Token válido por 1 hora

	_, err = qtx.CreateToken(r.Context(), sqlc.CreateTokenParams{
		UsuarioID:  user.ID,
		Token:      token,
		Tipo:       "RECUPERACION_PASSWORD",
		Expiracion: expiracion,
	})
	if err != nil {
		http.Redirect(w, r, "/recover-password?error=Error al procesar solicitud", http.StatusSeeOther)
		return
	}

	// Enviar correo
	baseURL := getBaseURL(r)
	err = auth.SendPasswordRecoveryEmail(baseURL, email, token)
	if err != nil {
		http.Redirect(w, r, "/recover-password?error=Error al enviar mail: "+err.Error(), http.StatusSeeOther)
		return
	}

	tx.Commit()
	http.Redirect(w, r, "/login?success=Se ha enviado el correo con instrucciones para restablecer tu contrasena.", http.StatusSeeOther)
}

// ShowResetPassword muestra el formulario de ingreso de nueva contraseña
func (h *AuthHandler) ShowResetPassword(w http.ResponseWriter, r *http.Request) {
	tokenVal := r.URL.Query().Get("token")
	if tokenVal == "" {
		http.Redirect(w, r, "/login?error=Token no provisto", http.StatusSeeOther)
		return
	}

	// Verificar si el token existe
	_, err := h.queries.GetTokenByValueAndTipo(r.Context(), sqlc.GetTokenByValueAndTipoParams{
		Token: tokenVal,
		Tipo:  "RECUPERACION_PASSWORD",
	})
	if err != nil {
		http.Redirect(w, r, "/login?error=Enlace de restablecimiento invalido o expirado", http.StatusSeeOther)
		return
	}

	data := map[string]interface{}{
		"Token": tokenVal,
	}
	RenderTemplate(w, r, "reset_password.html", data)
}

// ResetPassword procesa el cambio de contraseña con el token provisto
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	tokenVal := r.FormValue("token")
	password := r.FormValue("password")

	if tokenVal == "" || len(password) < 6 {
		http.Redirect(w, r, "/login?error=Datos invalidos o contrasena demasiado corta", http.StatusSeeOther)
		return
	}

	tokenObj, err := h.queries.GetTokenByValueAndTipo(r.Context(), sqlc.GetTokenByValueAndTipoParams{
		Token: tokenVal,
		Tipo:  "RECUPERACION_PASSWORD",
	})
	if err != nil {
		http.Redirect(w, r, "/login?error=Token invalido", http.StatusSeeOther)
		return
	}

	if tokenObj.Expiracion.Before(time.Now()) {
		_ = h.queries.DeleteToken(r.Context(), tokenObj.ID)
		http.Redirect(w, r, "/login?error=El enlace de recuperacion ha expirado", http.StatusSeeOther)
		return
	}

	// Hashear nueva contraseña
	hash, err := auth.HashPassword(password)
	if err != nil {
		http.Redirect(w, r, "/login?error=Error al procesar contrasena", http.StatusSeeOther)
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Redirect(w, r, "/login?error=Error de base de datos", http.StatusSeeOther)
		return
	}
	defer tx.Rollback()

	qtx := h.queries.WithTx(tx)

	// Actualizar contraseña
	err = qtx.UpdateUserPassword(r.Context(), sqlc.UpdateUserPasswordParams{
		PasswordHash: hash,
		ID:           tokenObj.UsuarioID,
	})
	if err != nil {
		http.Redirect(w, r, "/login?error=Error al cambiar contrasena", http.StatusSeeOther)
		return
	}

	// Limpiar token
	err = qtx.DeleteToken(r.Context(), tokenObj.ID)
	if err != nil {
		http.Redirect(w, r, "/login?error=Error al limpiar token", http.StatusSeeOther)
		return
	}

	tx.Commit()
	http.Redirect(w, r, "/login?success=Contrasena restablecida con exito. Inicia sesion.", http.StatusSeeOther)
}

// ChangePassword permite cambiar la contraseña desde el panel de control (usuario logueado)
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
		return
	}

	session := middleware.GetSession(r.Context())
	if session == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	actual := r.FormValue("password_actual")
	nuevo := r.FormValue("password_nuevo")

	if len(nuevo) < 6 {
		http.Redirect(w, r, "/dashboard?error=La nueva contrasena debe tener al menos 6 caracteres", http.StatusSeeOther)
		return
	}

	user, err := h.queries.GetUserById(r.Context(), session.UserID)
	if err != nil {
		http.Redirect(w, r, "/dashboard?error=Usuario no encontrado", http.StatusSeeOther)
		return
	}

	// Comprobar contraseña actual
	if !auth.CheckPasswordHash(actual, user.PasswordHash) {
		http.Redirect(w, r, "/dashboard?error=La contrasena actual es incorrecta", http.StatusSeeOther)
		return
	}

	// Hashear nueva contraseña
	hash, err := auth.HashPassword(nuevo)
	if err != nil {
		http.Redirect(w, r, "/dashboard?error=Error al hashear contrasena", http.StatusSeeOther)
		return
	}

	err = h.queries.UpdateUserPassword(r.Context(), sqlc.UpdateUserPasswordParams{
		PasswordHash: hash,
		ID:           user.ID,
	})
	if err != nil {
		http.Redirect(w, r, "/dashboard?error=Error al guardar contrasena", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/dashboard?success=Contrasena cambiada con exito", http.StatusSeeOther)
}

// Logout destruye la sesión y redirige al inicio
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/login?success=Sesion cerrada correctamente", http.StatusSeeOther)
}

// Helpers internos
func generateSecureToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}
