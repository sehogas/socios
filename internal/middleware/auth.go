package middleware

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/sehogas/socios3/db/sqlc"
	"github.com/sehogas/socios3/internal/auth"
)

type contextKey string

const SessionKey contextKey = "session"

type Middleware struct {
	queries *sqlc.Queries
}

func NewMiddleware(queries *sqlc.Queries) *Middleware {
	return &Middleware{queries: queries}
}

// GetSession obtiene la sesión guardada en el contexto de la petición
func GetSession(ctx context.Context) *auth.Session {
	sess, ok := ctx.Value(SessionKey).(*auth.Session)
	if !ok {
		return nil
	}
	return sess
}

// SessionLoader extrae la cookie de sesión, la descifra y la agrega al contexto
func (m *Middleware) SessionLoader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(auth.CookieName)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		session, err := auth.DecryptSession(cookie.Value)
		if err != nil {
			// Cookie alterada o inválida, la limpiamos
			auth.ClearSessionCookie(w)
			next.ServeHTTP(w, r)
			return
		}

		// Validar si el usuario sigue activo en la DB
		dbUser, err := m.queries.GetUserById(r.Context(), session.UserID)
		if err != nil || !dbUser.Activo.Valid || dbUser.Activo.Int64 == 0 {
			// Usuario inactivo o eliminado, removemos sesión
			auth.ClearSessionCookie(w)
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), SessionKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuth redirige al login si el usuario no tiene una sesión activa
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := GetSession(r.Context())
		if session == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole valida que el usuario pertenezca a los roles especificados
func (m *Middleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session := GetSession(r.Context())
			if session == nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			roleAllowed := false
			for _, rol := range roles {
				if session.Rol == rol {
					roleAllowed = true
					break
				}
			}

			if !roleAllowed {
				http.Error(w, "Acceso denegado: permisos insuficientes.", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireSocioActivo valida que, si es rol 'user', esté vinculado a un socio Activo.
// Los administradores tienen acceso sin esta validación.
func (m *Middleware) RequireSocioActivo(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := GetSession(r.Context())
		if session == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if session.Rol == "admin" {
			next.ServeHTTP(w, r)
			return
		}

		// Validar si el usuario está asociado a un socio activo
		socio, err := m.queries.GetSocioByEmail(r.Context(), sql.NullString{String: session.Email, Valid: true})
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Acceso denegado: no estás registrado como socio del club.", http.StatusForbidden)
				return
			}
			http.Error(w, "Error interno al validar socio.", http.StatusInternalServerError)
			return
		}

		if socio.Activo == 0 {
			http.Error(w, "Acceso denegado: tu cuenta de socio está inhabilitada. Contacta a administración.", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
