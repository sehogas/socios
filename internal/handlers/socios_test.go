package handlers_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/sehogas/socios/db/sqlc"
	"github.com/sehogas/socios/internal/handlers"
	"github.com/sehogas/socios/internal/testutil"
)

func TestPayQuotaFromCtaCte(t *testing.T) {
	db, queries, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup db: %v", err)
	}
	defer db.Close()

	handlers.SetDatabase(db, queries)
	h := handlers.NewSociosHandler(queries, db)

	ctx := context.Background()

	// 1. Crear un socio de prueba
	socio, err := queries.CreateSocio(ctx, sqlc.CreateSocioParams{
		Nombre:        "Juan",
		Apellido:      "Perez",
		TipoDocumento: sql.NullString{String: "DNI", Valid: true},
		NroDocumento:  "12345678",
		Clasificacion: "Titular",
		Estado:        "Aprobado",
		Activo:        1,
		Email:         sql.NullString{String: "juan@test.com", Valid: true},
	})
	if err != nil {
		t.Fatalf("create socio: %v", err)
	}

	// 2. Generar una cuota impaga de $5000 para el socio (esto inserta un DEBITO en la cuenta corriente)
	cuota, err := queries.CreateCuotaGenerada(ctx, sqlc.CreateCuotaGeneradaParams{
		SocioID:        socio.ID,
		Periodo:        "2026-06",
		MontoOriginal:  5000.00,
		MontoPendiente: 5000.00,
		Estado:         "Impaga",
	})
	if err != nil {
		t.Fatalf("create cuota: %v", err)
	}

	// Registrar el DÉBITO automático de la cuota en su Cta Cte (saldo actual: -$5000)
	_, err = queries.CreateTransaccionCtaCte(ctx, sqlc.CreateTransaccionCtaCteParams{
		SocioID:     socio.ID,
		Tipo:        "DEBITO",
		Monto:       5000.00,
		Fecha:       "2026-06-28",
		Descripcion: sql.NullString{String: "Cuota Social 2026-06", Valid: true},
	})
	if err != nil {
		t.Fatalf("create debit trans: %v", err)
	}

	t.Run("Saldo Insuficiente", func(t *testing.T) {
		// Intentamos pagar la cuota de $5000 cuando no se ha registrado ningún CRÉDITO.
		// El saldo neto actual es -$5000. Disponible = Saldo (-5000) + Cuota (5000) = $0.
		// Requerido = $5000. Debería rebotar por saldo insuficiente.
		req := httptest.NewRequest("POST", "/admin/socios/cuotas/pagar?id="+strconv.FormatInt(cuota.ID, 10)+"&socio_id="+strconv.FormatInt(socio.ID, 10), nil)
		rr := httptest.NewRecorder()

		h.PayQuotaFromCtaCte(rr, req)

		if rr.Code != http.StatusSeeOther {
			t.Errorf("esperado 303, obtenido %d", rr.Code)
		}

		if !strings.Contains(rr.Header().Get("Location"), "error=El socio") {
			t.Errorf("esperado redirección por saldo insuficiente, Location: %s", rr.Header().Get("Location"))
		}
	})

	t.Run("Saldo Suficiente", func(t *testing.T) {
		// Registramos dos movimientos de CRÉDITO para alcanzar saldo neto de $0 (saldar deuda de $5000)
		// Crédito 1: $1000 Efectivo
		_, err = queries.CreateTransaccionCtaCte(ctx, sqlc.CreateTransaccionCtaCteParams{
			SocioID:     socio.ID,
			Tipo:        "CREDITO",
			Monto:       1000.00,
			Fecha:       "2026-06-28",
			Descripcion: sql.NullString{String: "Pago efectivo", Valid: true},
		})
		if err != nil {
			t.Fatalf("create credit 1: %v", err)
		}

		// Crédito 2: $4000 Transferencia
		_, err = queries.CreateTransaccionCtaCte(ctx, sqlc.CreateTransaccionCtaCteParams{
			SocioID:     socio.ID,
			Tipo:        "CREDITO",
			Monto:       4000.00,
			Fecha:       "2026-06-28",
			Descripcion: sql.NullString{String: "Pago transferencia", Valid: true},
		})
		if err != nil {
			t.Fatalf("create credit 2: %v", err)
		}

		// Ahora el saldo neto es -$5000 + $1000 + $4000 = $0.
		// Disponible = Saldo (0) + Cuota (5000) = $5000. Requerido = $5000.
		// Debería permitir pagar la cuota de forma exitosa.
		req := httptest.NewRequest("POST", "/admin/socios/cuotas/pagar?id="+strconv.FormatInt(cuota.ID, 10)+"&socio_id="+strconv.FormatInt(socio.ID, 10), nil)
		rr := httptest.NewRecorder()

		h.PayQuotaFromCtaCte(rr, req)

		if rr.Code != http.StatusSeeOther {
			t.Errorf("esperado 303, obtenido %d", rr.Code)
		}

		if !strings.Contains(rr.Header().Get("Location"), "success=") {
			t.Errorf("esperado redirección exitosa, Location: %s", rr.Header().Get("Location"))
		}

		// Validar que la cuota quedó paga en la DB usando raw query
		var c sqlc.CuotasGenerada
		err = db.QueryRowContext(ctx, "SELECT id, socio_id, periodo, monto_original, monto_pendiente, estado FROM cuotas_generadas WHERE id = ?;", cuota.ID).
			Scan(&c.ID, &c.SocioID, &c.Periodo, &c.MontoOriginal, &c.MontoPendiente, &c.Estado)
		if err != nil {
			t.Fatalf("get cuota: %v", err)
		}

		if c.Estado != "Paga" {
			t.Errorf("esperado estado 'Paga', obtenido '%s'", c.Estado)
		}
		if c.MontoPendiente != 0 {
			t.Errorf("esperado monto pendiente 0, obtenido %.2f", c.MontoPendiente)
		}
	})
}
