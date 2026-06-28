package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sehogas/socios/db/sqlc"
	"github.com/sehogas/socios/internal/handlers"
	"github.com/sehogas/socios/internal/testutil"
)

func TestCuentasGeneral(t *testing.T) {
	db, queries, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup db: %v", err)
	}
	defer db.Close()

	handlers.SetDatabase(db, queries)
	h := handlers.NewCuentasHandler(queries, db)

	req := httptest.NewRequest("GET", "/admin/cuentas", nil)
	rr := httptest.NewRecorder()

	h.CuentasGeneral(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("esperado 200, obtenido %d", rr.Code)
	}
}

func TestCreateCajaTransaction(t *testing.T) {
	db, queries, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup db: %v", err)
	}
	defer db.Close()

	handlers.SetDatabase(db, queries)
	h := handlers.NewCuentasHandler(queries, db)

	t.Run("Monto Valido", func(t *testing.T) {
		form := url.Values{}
		form.Add("tipo", "INGRESO")
		form.Add("cuenta", "Efectivo")
		form.Add("monto", "1500.50")
		form.Add("fecha", "2026-06-28")
		form.Add("categoria", "Otros Ingresos")
		form.Add("descripcion", "Venta de rifas")

		req := httptest.NewRequest("POST", "/admin/cuentas/nueva-transaccion", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()

		h.CreateCajaTransaction(rr, req)

		if rr.Code != http.StatusSeeOther {
			t.Errorf("esperado 303, obtenido %d", rr.Code)
		}

		balance, err := queries.GetCajaBalanceByCuenta(context.Background(), "Efectivo")
		if err != nil {
			t.Fatalf("get balance: %v", err)
		}
		if balance != 1500.50 {
			t.Errorf("esperado balance de 1500.50, obtenido %.2f", balance)
		}
	})

	t.Run("Monto Invalido", func(t *testing.T) {
		form := url.Values{}
		form.Add("tipo", "EGRESO")
		form.Add("cuenta", "Efectivo")
		form.Add("monto", "-500")
		form.Add("fecha", "2026-06-28")
		form.Add("categoria", "Servicios")

		req := httptest.NewRequest("POST", "/admin/cuentas/nueva-transaccion", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()

		h.CreateCajaTransaction(rr, req)

		if rr.Code != http.StatusSeeOther {
			t.Errorf("esperado 303 redirect, obtenido %d", rr.Code)
		}

		if !strings.Contains(rr.Header().Get("Location"), "error=Monto") {
			t.Errorf("esperado redirección por error de monto, Location: %s", rr.Header().Get("Location"))
		}
	})
}

func TestTransferBetweenAccounts(t *testing.T) {
	db, queries, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup db: %v", err)
	}
	defer db.Close()

	handlers.SetDatabase(db, queries)
	h := handlers.NewCuentasHandler(queries, db)

	ctx := context.Background()
	// Cargar saldo inicial en efectivo
	_, err = queries.CreateTransaccionCaja(ctx, sqlc.CreateTransaccionCajaParams{
		Tipo:      "INGRESO",
		Cuenta:    "Efectivo",
		Monto:     2000.00,
		Fecha:     "2026-06-28",
		Categoria: "Aporte",
	})
	if err != nil {
		t.Fatalf("set initial cash: %v", err)
	}

	t.Run("Transferencia Exitosa", func(t *testing.T) {
		form := url.Values{}
		form.Add("desde", "Efectivo")
		form.Add("hacia", "Banco")
		form.Add("monto", "1200.00")
		form.Add("fecha", "2026-06-28")
		form.Add("descripcion", "Depósito semanal")

		req := httptest.NewRequest("POST", "/admin/cuentas/transferir", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()

		h.TransferBetweenAccounts(rr, req)

		if rr.Code != http.StatusSeeOther {
			t.Errorf("esperado 303, obtenido %d", rr.Code)
		}

		efectivo, _ := queries.GetCajaBalanceByCuenta(ctx, "Efectivo")
		banco, _ := queries.GetCajaBalanceByCuenta(ctx, "Banco")

		if efectivo != 800.00 {
			t.Errorf("esperado Efectivo = 800, obtenido %.2f", efectivo)
		}
		if banco != 1200.00 {
			t.Errorf("esperado Banco = 1200, obtenido %.2f", banco)
		}
	})

	t.Run("Saldo Insuficiente", func(t *testing.T) {
		form := url.Values{}
		form.Add("desde", "Efectivo")
		form.Add("hacia", "Banco")
		form.Add("monto", "5000.00") // Mayor al saldo actual
		form.Add("fecha", "2026-06-28")

		req := httptest.NewRequest("POST", "/admin/cuentas/transferir", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()

		h.TransferBetweenAccounts(rr, req)

		if rr.Code != http.StatusSeeOther {
			t.Errorf("esperado 303, obtenido %d", rr.Code)
		}

		if !strings.Contains(rr.Header().Get("Location"), "error=Saldo") {
			t.Errorf("esperado redirección por saldo insuficiente, Location: %s", rr.Header().Get("Location"))
		}
	})
}
