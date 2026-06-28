package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sehogas/socios/db/sqlc"
)

type CuentasHandler struct {
	queries *sqlc.Queries
	db      *sql.DB
}

func NewCuentasHandler(queries *sqlc.Queries, db *sql.DB) *CuentasHandler {
	return &CuentasHandler{queries: queries, db: db}
}

// CuentasGeneral muestra el balance de las cuentas y el libro diario de caja
func (h *CuentasHandler) CuentasGeneral(w http.ResponseWriter, r *http.Request) {
	// Calcular saldos de cada cuenta
	saldoEfectivo, _ := h.queries.GetCajaBalanceByCuenta(r.Context(), "Efectivo")
	saldoBanco, _ := h.queries.GetCajaBalanceByCuenta(r.Context(), "Banco")
	saldoMP, _ := h.queries.GetCajaBalanceByCuenta(r.Context(), "MercadoPago")
	
	totalGeneral := saldoEfectivo + saldoBanco + saldoMP

	// Obtener todos los movimientos de caja
	movimientos, err := h.queries.ListTransaccionesCaja(r.Context())
	if err != nil {
		http.Error(w, "Error al listar movimientos de caja: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Obtener valores de cuota históricos
	valoresCuota, _ := h.queries.ListCuotasValores(r.Context())

	// Obtener resumen por categorías para el Balance Financiero
	var ingresosCats []sqlc.GetCajaSummaryByCategoryRow
	var egresosCats []sqlc.GetCajaSummaryByCategoryRow
	var totalIngresos float64
	var totalEgresos float64

	summary, _ := h.queries.GetCajaSummaryByCategory(r.Context())
	for _, row := range summary {
		if row.Tipo == "INGRESO" {
			ingresosCats = append(ingresosCats, row)
			totalIngresos += row.Total
		} else if row.Tipo == "EGRESO" {
			egresosCats = append(egresosCats, row)
			totalEgresos += row.Total
		}
	}

	data := map[string]interface{}{
		"SaldoEfectivo":      saldoEfectivo,
		"SaldoBanco":         saldoBanco,
		"SaldoMP":            saldoMP,
		"TotalGeneral":       totalGeneral,
		"Movimientos":        movimientos,
		"FechaHoy":           time.Now().Format("2006-01-02"),
		"VigenciaHoy":        time.Now().Format("2006-01"),
		"ValoresCuota":       valoresCuota,
		"IngresosCategorias": ingresosCats,
		"EgresosCategorias":  egresosCats,
		"TotalIngresos":      totalIngresos,
		"TotalEgresos":       totalEgresos,
		"SuperavitDeficit":   totalIngresos - totalEgresos,
	}
	RenderTemplate(w, r, "admin/cuentas.html", data)
}

// CreateCajaTransaction registra un movimiento manual en la tesorería (ingreso o egreso)
func (h *CuentasHandler) CreateCajaTransaction(w http.ResponseWriter, r *http.Request) {
	tipo := r.FormValue("tipo") // INGRESO o EGRESO
	cuenta := r.FormValue("cuenta") // Efectivo, Banco, MercadoPago
	montoStr := r.FormValue("monto")
	fecha := r.FormValue("fecha")
	categoria := r.FormValue("categoria")
	descripcion := r.FormValue("descripcion")

	monto, err := strconv.ParseFloat(montoStr, 64)
	if err != nil || monto <= 0 {
		http.Redirect(w, r, "/admin/cuentas?error=Monto invalido", http.StatusSeeOther)
		return
	}

	var desc sql.NullString
	if descripcion != "" {
		desc = sql.NullString{String: descripcion, Valid: true}
	}

	_, err = h.queries.CreateTransaccionCaja(r.Context(), sqlc.CreateTransaccionCajaParams{
		Tipo:        tipo,
		Cuenta:      cuenta,
		Monto:       monto,
		Fecha:       fecha,
		Categoria:   categoria,
		Descripcion: desc,
	})
	if err != nil {
		http.Redirect(w, r, "/admin/cuentas?error=Error al guardar movimiento: "+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/cuentas?success=Movimiento de caja registrado exitosamente", http.StatusSeeOther)
}

// GenerateMonthlyQuotas genera las cuotas mensuales para todos los socios aprobados y activos
func (h *CuentasHandler) GenerateMonthlyQuotas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
		return
	}

	periodo := time.Now().Format("2006-01") // Formato YYYY-MM
	
	// Obtener socios
	socios, err := h.queries.ListSocios(r.Context())
	if err != nil {
		http.Redirect(w, r, "/dashboard?error=Error al obtener listado de socios", http.StatusSeeOther)
		return
	}

	// Identificar las clasificaciones de socios activos y aprobados
	clasificacionesRequeridas := make(map[string]bool)
	for _, socio := range socios {
		if socio.Estado == "Aprobado" && socio.Activo != 0 {
			clasificacionesRequeridas[socio.Clasificacion] = true
		}
	}

	// Verificar que todas las clasificaciones requeridas tengan valor de cuota configurado
	for cls := range clasificacionesRequeridas {
		_, err := h.queries.GetCuotaValorByClasificacionAndPeriodo(r.Context(), sqlc.GetCuotaValorByClasificacionAndPeriodoParams{
			Clasificacion:   cls,
			VigenciaInicial: periodo,
		})
		if err != nil {
			msg := fmt.Sprintf("No se pueden generar las cuotas. Falta configurar el valor de la cuota para la clasificación '%s' para el período %s o anterior.", cls, periodo)
			http.Redirect(w, r, "/dashboard?error="+msg, http.StatusSeeOther)
			return
		}
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Redirect(w, r, "/dashboard?error=Error de base de datos", http.StatusSeeOther)
		return
	}
	defer tx.Rollback()

	qtx := h.queries.WithTx(tx)

	cuotasGeneradasCount := 0

	for _, socio := range socios {
		// Solo generar para socios aprobados y con cuenta activa (no suspendidos)
		if socio.Estado != "Aprobado" || socio.Activo == 0 {
			continue
		}

		// Verificar si ya existe cuota para este periodo
		_, err := qtx.GetCuotaGeneradaBySocioAndPeriodo(r.Context(), sqlc.GetCuotaGeneradaBySocioAndPeriodoParams{
			SocioID: socio.ID,
			Periodo: periodo,
		})
		if err == nil {
			// Ya existe cuota para este socio en este mes
			continue
		}

		// Determinar monto de la cuota
		montoCuota := 1000.0
		val, err := qtx.GetCuotaValorByClasificacionAndPeriodo(r.Context(), sqlc.GetCuotaValorByClasificacionAndPeriodoParams{
			Clasificacion:   socio.Clasificacion,
			VigenciaInicial: periodo,
		})
		if err == nil {
			montoCuota = val.Monto
		} else {
			// Fallback a legacy cuotas_config o default según clasificación
			legacyCat := "Activo"
			if socio.Clasificacion == "Adherente" {
				legacyCat = "Adherente"
			}
			conf, err := qtx.GetCuotaConfigByCategoria(r.Context(), legacyCat)
			if err == nil {
				montoCuota = conf.Monto
			} else {
				// Valores por defecto tradicionales
				switch socio.Clasificacion {
				case "Honorario", "Vitalicio":
					montoCuota = 0.0
				case "Adherente":
					montoCuota = 500.0
				default:
					montoCuota = 1000.0
				}
			}
		}

		// 1. Crear la cuota generada
		_, err = qtx.CreateCuotaGenerada(r.Context(), sqlc.CreateCuotaGeneradaParams{
			SocioID:        socio.ID,
			Periodo:        periodo,
			MontoOriginal:  montoCuota,
			MontoPendiente: montoCuota,
			Estado:         "Impaga",
		})
		if err != nil {
			http.Redirect(w, r, "/dashboard?error=Error al generar cuota: "+err.Error(), http.StatusSeeOther)
			return
		}

		// 2. Registrar el Débito (deuda) en la Cuenta Corriente del Socio
		_, err = qtx.CreateTransaccionCtaCte(r.Context(), sqlc.CreateTransaccionCtaCteParams{
			SocioID:     socio.ID,
			Tipo:        "DEBITO",
			Monto:       montoCuota,
			Fecha:       time.Now().Format("2006-01-02"),
			Descripcion: sql.NullString{String: fmt.Sprintf("Cargo de cuota social periodo %s", periodo), Valid: true},
		})
		if err != nil {
			http.Redirect(w, r, "/dashboard?error=Error al impactar cuenta corriente: "+err.Error(), http.StatusSeeOther)
			return
		}

		cuotasGeneradasCount++
	}

	tx.Commit()

	msg := fmt.Sprintf("Se generaron exitosamente %d cuotas para el periodo %s", cuotasGeneradasCount, periodo)
	http.Redirect(w, r, "/dashboard?success="+msg, http.StatusSeeOther)
}

// UpdateCuotaValor registra o actualiza un valor de cuota para una clasificación y vigencia inicial dada
func (h *CuentasHandler) UpdateCuotaValor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
		return
	}

	clasificacion := r.FormValue("clasificacion")
	montoStr := r.FormValue("monto")
	vigencia := r.FormValue("vigencia_inicial") // Formato YYYY-MM

	monto, err := strconv.ParseFloat(montoStr, 64)
	if err != nil || monto < 0 {
		http.Redirect(w, r, "/admin/cuentas?error=Monto invalido", http.StatusSeeOther)
		return
	}

	// Validar formato YYYY-MM
	if len(vigencia) != 7 || vigencia[4] != '-' {
		http.Redirect(w, r, "/admin/cuentas?error=Formato de vigencia invalido (debe ser AAAA-MM)", http.StatusSeeOther)
		return
	}

	_, err = h.queries.CreateOrUpdateCuotaValor(r.Context(), sqlc.CreateOrUpdateCuotaValorParams{
		Clasificacion:   clasificacion,
		Monto:           monto,
		VigenciaInicial: vigencia,
	})
	if err != nil {
		http.Redirect(w, r, "/admin/cuentas?error=Error al guardar valor de cuota: "+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/cuentas?success=Valor de cuota social guardado exitosamente", http.StatusSeeOther)
}

// TransferBetweenAccounts realiza una transferencia de dinero entre cuentas de la institución (Efectivo, Banco, MercadoPago)
func (h *CuentasHandler) TransferBetweenAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
		return
	}

	desde := r.FormValue("desde")
	hacia := r.FormValue("hacia")
	montoStr := r.FormValue("monto")
	fecha := r.FormValue("fecha")
	descripcion := r.FormValue("descripcion")

	monto, err := strconv.ParseFloat(montoStr, 64)
	if err != nil || monto <= 0 {
		http.Redirect(w, r, "/admin/cuentas?error=Monto de transferencia invalido", http.StatusSeeOther)
		return
	}

	if desde == hacia {
		http.Redirect(w, r, "/admin/cuentas?error=La cuenta de origen y destino deben ser distintas", http.StatusSeeOther)
		return
	}

	// Validar que las cuentas sean válidas
	validAccounts := map[string]bool{"Efectivo": true, "Banco": true, "MercadoPago": true}
	if !validAccounts[desde] || !validAccounts[hacia] {
		http.Redirect(w, r, "/admin/cuentas?error=Cuentas de transferencia invalidas", http.StatusSeeOther)
		return
	}

	// Iniciar transacción de base de datos
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Redirect(w, r, "/admin/cuentas?error=Error de base de datos al iniciar transferencia", http.StatusSeeOther)
		return
	}
	defer tx.Rollback()

	qtx := h.queries.WithTx(tx)

	// Validar que haya saldo suficiente en la cuenta de origen
	saldoOrigen, err := qtx.GetCajaBalanceByCuenta(r.Context(), desde)
	if err != nil {
		http.Redirect(w, r, "/admin/cuentas?error=Error al verificar saldo de origen", http.StatusSeeOther)
		return
	}

	if saldoOrigen < monto {
		http.Redirect(w, r, fmt.Sprintf("/admin/cuentas?error=Saldo insuficiente en la cuenta de origen '%s' (Disponible: $%.2f, Requerido: $%.2f)", desde, saldoOrigen, monto), http.StatusSeeOther)
		return
	}

	descStr := fmt.Sprintf("Transferencia interna de %s a %s", desde, hacia)
	if descripcion != "" {
		descStr = descStr + " - " + descripcion
	}

	// 1. Registrar el EGRESO de la cuenta de origen
	_, err = qtx.CreateTransaccionCaja(r.Context(), sqlc.CreateTransaccionCajaParams{
		Tipo:        "EGRESO",
		Cuenta:      desde,
		Monto:       monto,
		Fecha:       fecha,
		Categoria:   "Transferencia Interna",
		Descripcion: sql.NullString{String: descStr, Valid: true},
	})
	if err != nil {
		http.Redirect(w, r, "/admin/cuentas?error=Error al debitar de la cuenta de origen: "+err.Error(), http.StatusSeeOther)
		return
	}

	// 2. Registrar el INGRESO en la cuenta de destino
	_, err = qtx.CreateTransaccionCaja(r.Context(), sqlc.CreateTransaccionCajaParams{
		Tipo:        "INGRESO",
		Cuenta:      hacia,
		Monto:       monto,
		Fecha:       fecha,
		Categoria:   "Transferencia Interna",
		Descripcion: sql.NullString{String: descStr, Valid: true},
	})
	if err != nil {
		http.Redirect(w, r, "/admin/cuentas?error=Error al acreditar en la cuenta de destino: "+err.Error(), http.StatusSeeOther)
		return
	}

	tx.Commit()

	http.Redirect(w, r, fmt.Sprintf("/admin/cuentas?success=Transferencia de $%.2f desde %s hacia %s realizada correctamente", monto, desde, hacia), http.StatusSeeOther)
}
