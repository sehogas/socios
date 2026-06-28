package middleware_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sehogas/socios/db/sqlc"
	"github.com/sehogas/socios/internal/auth"
	"github.com/sehogas/socios/internal/middleware"
	"github.com/sehogas/socios/internal/testutil"
)

func TestSessionLoader(t *testing.T) {
	db, queries, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	// Crear un usuario activo
	user, err := queries.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        "admin@test.com",
		PasswordHash: "hashed",
		Rol:          "admin",
		Activo:       sql.NullInt64{Int64: 1, Valid: true},
		Verificado:   1,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	m := middleware.NewMiddleware(queries)

	// Handler de prueba que valida la sesión
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := middleware.GetSession(r.Context())
		if sess == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if sess.UserID != user.ID {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	t.Run("Sin Cookie de Sesion", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		m.SessionLoader(testHandler).ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("esperado 401, obtenido %d", rr.Code)
		}
	})

	t.Run("Con Cookie de Sesion Valida", func(t *testing.T) {
		sess := &auth.Session{
			UserID:    user.ID,
			Email:     user.Email,
			Rol:       user.Rol,
			CreatedAt: time.Now(),
		}
		cookieValue, err := auth.EncryptSession(sess)
		if err != nil {
			t.Fatalf("encrypt session: %v", err)
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  auth.CookieName,
			Value: cookieValue,
		})
		rr := httptest.NewRecorder()

		m.SessionLoader(testHandler).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("esperado 200, obtenido %d", rr.Code)
		}
	})

	t.Run("Con Cookie de Sesion de Usuario Inactivo", func(t *testing.T) {
		// Crear usuario inactivo
		inactiveUser, err := queries.CreateUser(ctx, sqlc.CreateUserParams{
			Email:        "inactive@test.com",
			PasswordHash: "hashed",
			Rol:          "user",
			Activo:       sql.NullInt64{Int64: 0, Valid: true},
			Verificado:   1,
		})
		if err != nil {
			t.Fatalf("create inactive user: %v", err)
		}

		sess := &auth.Session{
			UserID:    inactiveUser.ID,
			Email:     inactiveUser.Email,
			Rol:       inactiveUser.Rol,
			CreatedAt: time.Now(),
		}
		cookieValue, err := auth.EncryptSession(sess)
		if err != nil {
			t.Fatalf("encrypt session: %v", err)
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  auth.CookieName,
			Value: cookieValue,
		})
		rr := httptest.NewRecorder()

		m.SessionLoader(testHandler).ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("esperado 401, obtenido %d", rr.Code)
		}
	})
}
