package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/sehogas/socios3/db/sqlc"
)

type SociosHandler struct {
	queries *sqlc.Queries
	db      *sql.DB
}

func NewSociosHandler(queries *sqlc.Queries, db *sql.DB) *SociosHandler {
	return &SociosHandler{queries: queries, db: db}
}

// ListSocios muestra la lista de todos los socios
func (h *SociosHandler) ListSocios(w http.ResponseWriter, r *http.Request) {
	socios, err := h.queries.ListSocios(r.Context())
	if err != nil {
		http.Error(w, "Error al listar socios: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Socios": socios,
	}
	RenderTemplate(w, r, "admin/socios_list.html", data)
}

// ShowNewSocioForm muestra el formulario para cargar una nueva solicitud
func (h *SociosHandler) ShowNewSocioForm(w http.ResponseWriter, r *http.Request) {
	todos, err := h.queries.ListSocios(r.Context())
	var titulares []sqlc.Socio
	if err == nil {
		for _, s := range todos {
			if s.Clasificacion == "Titular" {
				titulares = append(titulares, s)
			}
		}
	}
	data := map[string]interface{}{
		"Titulares": titulares,
	}
	RenderTemplate(w, r, "admin/socio_form.html", data)
}

// CreateSocio procesa la inserción de una nueva ficha
func (h *SociosHandler) CreateSocio(w http.ResponseWriter, r *http.Request) {
	params := h.parseSocioParams(r)

	// Validaciones obligatorias
	if !params.TipoDocumento.Valid || params.TipoDocumento.String == "" {
		http.Redirect(w, r, "/admin/socios/nuevo?error=El tipo de documento es obligatorio", http.StatusSeeOther)
		return
	}

	if params.Email.Valid && params.Email.String != "" {
		if _, err := mail.ParseAddress(params.Email.String); err != nil {
			http.Redirect(w, r, "/admin/socios/nuevo?error=El correo electronico ingresado no es valido", http.StatusSeeOther)
			return
		}
	}

	// Si es creado como aprobado y no tiene número manual, le asignamos el correlativo
	if params.Estado == "Aprobado" && (!params.NumeroSocio.Valid || params.NumeroSocio.String == "") {
		nextNum := h.generateNextNumeroSocio(r.Context())
		params.NumeroSocio = sql.NullString{String: nextNum, Valid: true}
		params.FechaAprobacion = sql.NullString{String: time.Now().Format("2006-01-02"), Valid: true}
	}

	_, err := h.queries.CreateSocio(r.Context(), params)
	if err != nil {
		http.Redirect(w, r, "/admin/socios/nuevo?error=Error al guardar socio: "+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/socios?success=Planilla de inscripcion guardada correctamente", http.StatusSeeOther)
}

// ShowEditSocioForm muestra el formulario de edición cargando los datos existentes
func (h *SociosHandler) ShowEditSocioForm(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/socios?error=ID invalido", http.StatusSeeOther)
		return
	}

	socio, err := h.queries.GetSocioById(r.Context(), id)
	if err != nil {
		http.Redirect(w, r, "/admin/socios?error=Socio no encontrado", http.StatusSeeOther)
		return
	}

	todos, err := h.queries.ListSocios(r.Context())
	var titulares []sqlc.Socio
	if err == nil {
		for _, s := range todos {
			if s.Clasificacion == "Titular" && s.ID != id {
				titulares = append(titulares, s)
			}
		}
	}

	data := map[string]interface{}{
		"Socio":     socio,
		"Titulares": titulares,
	}
	RenderTemplate(w, r, "admin/socio_form.html", data)
}

// UpdateSocio guarda los datos modificados del socio
func (h *SociosHandler) UpdateSocio(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/socios?error=ID invalido", http.StatusSeeOther)
		return
	}

	// Validar si existe
	socio, err := h.queries.GetSocioById(r.Context(), id)
	if err != nil {
		http.Redirect(w, r, "/admin/socios?error=Socio no encontrado", http.StatusSeeOther)
		return
	}

	params := h.parseSocioParams(r)

	// Validaciones obligatorias
	if !params.TipoDocumento.Valid || params.TipoDocumento.String == "" {
		http.Redirect(w, r, fmt.Sprintf("/admin/socios/editar?id=%d&error=El tipo de documento es obligatorio", id), http.StatusSeeOther)
		return
	}

	if params.Email.Valid && params.Email.String != "" {
		if _, err := mail.ParseAddress(params.Email.String); err != nil {
			http.Redirect(w, r, fmt.Sprintf("/admin/socios/editar?id=%d&error=El correo electronico ingresado no es valido", id), http.StatusSeeOther)
			return
		}
	}

	// Lógica de transición de estado
	nuevoEstado := r.FormValue("estado")
	nuevoNumero := r.FormValue("numero_socio")
	
	// Si pasa a Aprobado y no tenía número
	var numeroFinal sql.NullString
	var fechaAprobacion sql.NullString
	if nuevoEstado == "Aprobado" {
		if nuevoNumero != "" {
			numeroFinal = sql.NullString{String: nuevoNumero, Valid: true}
		} else if socio.NumeroSocio.Valid && socio.NumeroSocio.String != "" {
			numeroFinal = socio.NumeroSocio
		} else {
			numeroFinal = sql.NullString{String: h.generateNextNumeroSocio(r.Context()), Valid: true}
		}
		fechaAprobacion = sql.NullString{String: time.Now().Format("2006-01-02"), Valid: true}
	} else {
		numeroFinal = sql.NullString{Valid: false}
		fechaAprobacion = sql.NullString{Valid: false}
	}

	// Iniciar transacción para actualizar tanto el socio básico como sus estados
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Redirect(w, r, "/admin/socios?error=Error de base de datos", http.StatusSeeOther)
		return
	}
	defer tx.Rollback()

	qtx := h.queries.WithTx(tx)

	// Actualizar datos del socio
	err = qtx.UpdateSocio(r.Context(), sqlc.UpdateSocioParams{
		ID:              id,
		Nombre:          params.Nombre,
		Apellido:        params.Apellido,
		LugarNacimiento: params.LugarNacimiento,
		FechaNacimiento: params.FechaNacimiento,
		Nacionalidad:    params.Nacionalidad,
		EstadoCivil:     params.EstadoCivil,
		TipoDocumento:   params.TipoDocumento,
		NroDocumento:    params.NroDocumento,
		Profesion:       params.Profesion,
		LugarTrabajo:    params.LugarTrabajo,
		Domicilio:       params.Domicilio,
		Telefono:        params.Telefono,
		Email:           params.Email,
		Clasificacion:   params.Clasificacion,
		TitularID:       params.TitularID,
	})
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/socios/editar?id=%d&error=Error al actualizar: %s", id, err.Error()), http.StatusSeeOther)
		return
	}

	// Actualizar estado y número
	err = qtx.UpdateSocioStatus(r.Context(), sqlc.UpdateSocioStatusParams{
		ID:              id,
		Estado:          nuevoEstado,
		NumeroSocio:     numeroFinal,
		FechaAprobacion: fechaAprobacion,
	})
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/socios/editar?id=%d&error=Error al actualizar estado: %s", id, err.Error()), http.StatusSeeOther)
		return
	}

	// Actualizar activo
	activoVal, _ := strconv.Atoi(r.FormValue("activo"))
	err = qtx.UpdateSocioActive(r.Context(), sqlc.UpdateSocioActiveParams{
		ID:     id,
		Activo: int64(activoVal),
	})
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/socios/editar?id=%d&error=Error al actualizar activo: %s", id, err.Error()), http.StatusSeeOther)
		return
	}

	tx.Commit()
	http.Redirect(w, r, "/admin/socios?success=Socio actualizado correctamente", http.StatusSeeOther)
}

// ApproveSocio aprueba de forma rápida un socio y le asigna número
func (h *SociosHandler) ApproveSocio(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/socios?error=ID invalido", http.StatusSeeOther)
		return
	}

	nextNum := h.generateNextNumeroSocio(r.Context())
	err = h.queries.UpdateSocioStatus(r.Context(), sqlc.UpdateSocioStatusParams{
		ID:              id,
		Estado:          "Aprobado",
		NumeroSocio:     sql.NullString{String: nextNum, Valid: true},
		FechaAprobacion: sql.NullString{String: time.Now().Format("2006-01-02"), Valid: true},
	})
	if err != nil {
		http.Redirect(w, r, "/admin/socios?error=Error al aprobar socio: "+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/socios?success=Socio aprobado con Nro "+nextNum, http.StatusSeeOther)
}

// RejectSocio rechaza una solicitud de inscripción
func (h *SociosHandler) RejectSocio(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/socios?error=ID invalido", http.StatusSeeOther)
		return
	}

	err = h.queries.UpdateSocioStatus(r.Context(), sqlc.UpdateSocioStatusParams{
		ID:              id,
		Estado:          "Rechazado",
		NumeroSocio:     sql.NullString{Valid: false},
		FechaAprobacion: sql.NullString{Valid: false},
	})
	if err != nil {
		http.Redirect(w, r, "/admin/socios?error=Error al rechazar socio: "+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/socios?success=Solicitud rechazada", http.StatusSeeOther)
}

// ToggleActivo inhabilita o habilita el acceso de un socio
func (h *SociosHandler) ToggleActivo(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/socios?error=ID invalido", http.StatusSeeOther)
		return
	}

	activoVal, err := strconv.ParseInt(r.URL.Query().Get("activo"), 10, 64)
	if err != nil {
		activoVal = 1
	}

	err = h.queries.UpdateSocioActive(r.Context(), sqlc.UpdateSocioActiveParams{
		ID:     id,
		Activo: activoVal,
	})
	if err != nil {
		http.Redirect(w, r, "/admin/socios?error=Error al cambiar acceso de socio: "+err.Error(), http.StatusSeeOther)
		return
	}

	msg := "Socio habilitado"
	if activoVal == 0 {
		msg = "Socio inhabilitado"
	}
	http.Redirect(w, r, "/admin/socios?success="+msg, http.StatusSeeOther)
}

// SocioDetail muestra la ficha completa y cuenta corriente del socio
func (h *SociosHandler) SocioDetail(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/socios?error=ID invalido", http.StatusSeeOther)
		return
	}

	socio, err := h.queries.GetSocioById(r.Context(), id)
	if err != nil {
		http.Redirect(w, r, "/admin/socios?error=Socio no encontrado", http.StatusSeeOther)
		return
	}

	// Obtener movimientos y saldo
	ctaCte, _ := h.queries.ListTransaccionesCtaCteBySocio(r.Context(), id)
	saldo, _ := h.queries.GetSaldoSocio(r.Context(), id)
	cuotas, _ := h.queries.ListCuotasGeneradasBySocio(r.Context(), id)

	var titular *sqlc.Socio
	if socio.Clasificacion == "Adherente" && socio.TitularID.Valid {
		t, err := h.queries.GetSocioById(r.Context(), socio.TitularID.Int64)
		if err == nil {
			titular = &t
		}
	}

	data := map[string]interface{}{
		"Socio":    socio,
		"CtaCte":   ctaCte,
		"Saldo":    saldo,
		"Cuotas":   cuotas,
		"FechaHoy": time.Now().Format("2006-01-02"),
		"Titular":  titular,
	}
	RenderTemplate(w, r, "admin/socio_detail.html", data)
}

// CreateCtaCteTransaction agrega un movimiento a la cuenta corriente del socio
func (h *SociosHandler) CreateCtaCteTransaction(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/socios?error=ID invalido", http.StatusSeeOther)
		return
	}

	tipo := r.FormValue("tipo")
	montoStr := r.FormValue("monto")
	fecha := r.FormValue("fecha")
	descripcion := r.FormValue("descripcion")
	cuenta := r.FormValue("cuenta") // Si es crédito, puede impactar caja

	monto, err := strconv.ParseFloat(montoStr, 64)
	if err != nil || monto <= 0 {
		http.Redirect(w, r, fmt.Sprintf("/admin/socios/detalle?id=%d&error=Monto invalido", id), http.StatusSeeOther)
		return
	}

	// Transacción de base de datos para asegurar consistencia Cta Cte y Caja
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/socios/detalle?id=%d&error=Error al conectar a DB", id), http.StatusSeeOther)
		return
	}
	defer tx.Rollback()

	qtx := h.queries.WithTx(tx)

	var cajaID sql.NullInt64

	// Si es crédito y se seleccionó una cuenta de caja válida, registramos el ingreso real en caja
	if tipo == "CREDITO" && cuenta != "NINGUNA" && cuenta != "" {
		categoriaCaja := "Cobro Cuota"
		if strings.Contains(strings.ToLower(descripcion), "adelanto") {
			categoriaCaja = "Adelanto"
		}
		cajaTx, err := qtx.CreateTransaccionCaja(r.Context(), sqlc.CreateTransaccionCajaParams{
			Tipo:        "INGRESO",
			Cuenta:      cuenta,
			Monto:       monto,
			Fecha:       fecha,
			Categoria:   categoriaCaja,
			Descripcion: sql.NullString{String: fmt.Sprintf("Socio ID %d - %s", id, descripcion), Valid: true},
		})
		if err != nil {
			http.Redirect(w, r, fmt.Sprintf("/admin/socios/detalle?id=%d&error=Error al registrar ingreso en caja: %s", id, err.Error()), http.StatusSeeOther)
			return
		}
		cajaID = sql.NullInt64{Int64: cajaTx.ID, Valid: true}
	}

	// Registrar transacción en Cuenta Corriente
	_, err = qtx.CreateTransaccionCtaCte(r.Context(), sqlc.CreateTransaccionCtaCteParams{
		SocioID:           id,
		Tipo:              tipo,
		Monto:             monto,
		Fecha:             fecha,
		Descripcion:       sql.NullString{String: descripcion, Valid: true},
		TransaccionCajaID: cajaID,
	})
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/socios/detalle?id=%d&error=Error al registrar cuenta corriente: %s", id, err.Error()), http.StatusSeeOther)
		return
	}

	tx.Commit()
	http.Redirect(w, r, fmt.Sprintf("/admin/socios/detalle?id=%d&success=Movimiento registrado exitosamente", id), http.StatusSeeOther)
}

// PayQuotaFromCtaCte descuenta del saldo a favor del socio para pagar una cuota pendiente
func (h *SociosHandler) PayQuotaFromCtaCte(w http.ResponseWriter, r *http.Request) {
	quotaIdStr := r.URL.Query().Get("id")
	quotaID, err := strconv.ParseInt(quotaIdStr, 10, 64)
	socioIdStr := r.URL.Query().Get("socio_id")
	socioID, err2 := strconv.ParseInt(socioIdStr, 10, 64)

	if err != nil || err2 != nil {
		http.Redirect(w, r, "/admin/socios?error=ID invalido", http.StatusSeeOther)
		return
	}

	// Obtener cuota
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/socios/detalle?id=%d&error=Error de conexion", socioID), http.StatusSeeOther)
		return
	}
	defer tx.Rollback()

	qtx := h.queries.WithTx(tx)

	// Buscar cuota
	var cuota sqlc.CuotasGenerada
	err = tx.QueryRowContext(r.Context(), "SELECT id, socio_id, periodo, monto_original, monto_pendiente, estado FROM cuotas_generadas WHERE id = ?;", quotaID).
		Scan(&cuota.ID, &cuota.SocioID, &cuota.Periodo, &cuota.MontoOriginal, &cuota.MontoPendiente, &cuota.Estado)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/socios/detalle?id=%d&error=Cuota no encontrada", socioID), http.StatusSeeOther)
		return
	}

	// Obtener saldo del socio
	saldo, err := qtx.GetSaldoSocio(r.Context(), socioID)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/socios/detalle?id=%d&error=Error al obtener saldo", socioID), http.StatusSeeOther)
		return
	}

	// Si el saldo es menor al monto pendiente de la cuota, no se puede pagar
	if saldo < cuota.MontoPendiente {
		http.Redirect(w, r, fmt.Sprintf("/admin/socios/detalle?id=%d&error=El socio no tiene suficiente saldo a favor en su Cuenta Corriente (Saldo: $%.2f, Necesita: $%.2f)", socioID, saldo, cuota.MontoPendiente), http.StatusSeeOther)
		return
	}

	montoAPagar := cuota.MontoPendiente

	// Crear transacción de Débito (consumo de crédito) en la Cta Cte
	_, err = qtx.CreateTransaccionCtaCte(r.Context(), sqlc.CreateTransaccionCtaCteParams{
		SocioID:     socioID,
		Tipo:        "DEBITO",
		Monto:       montoAPagar,
		Fecha:       time.Now().Format("2006-06-02"),
		Descripcion: sql.NullString{String: fmt.Sprintf("Consumo saldo para pagar cuota periodo %s", cuota.Periodo), Valid: true},
	})
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/socios/detalle?id=%d&error=Error al registrar debito en cta cte: %s", socioID, err.Error()), http.StatusSeeOther)
		return
	}

	// Actualizar la cuota a Paga y pendiente = 0
	err = qtx.UpdateCuotaGeneradaMontoPendiente(r.Context(), sqlc.UpdateCuotaGeneradaMontoPendienteParams{
		MontoPendiente: 0,
		Estado:         "Paga",
		ID:             quotaID,
	})
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/socios/detalle?id=%d&error=Error al actualizar cuota: %s", socioID, err.Error()), http.StatusSeeOther)
		return
	}

	tx.Commit()
	http.Redirect(w, r, fmt.Sprintf("/admin/socios/detalle?id=%d&success=Cuota periodo %s pagada con exito usando saldo Cta Cte", socioID, cuota.Periodo), http.StatusSeeOther)
}

// Helpers internos
func (h *SociosHandler) parseSocioParams(r *http.Request) sqlc.CreateSocioParams {
	emailVal := strings.TrimSpace(r.FormValue("email"))
	var email sql.NullString
	if emailVal != "" {
		email = sql.NullString{String: emailVal, Valid: true}
	}

	numVal := strings.TrimSpace(r.FormValue("numero_socio"))
	var num sql.NullString
	if numVal != "" {
		num = sql.NullString{String: numVal, Valid: true}
	}

	telVal := strings.TrimSpace(r.FormValue("telefono"))
	var tel sql.NullString
	if telVal != "" {
		tel = sql.NullString{String: telVal, Valid: true}
	}

	domVal := strings.TrimSpace(r.FormValue("domicilio"))
	var dom sql.NullString
	if domVal != "" {
		dom = sql.NullString{String: domVal, Valid: true}
	}

	lugarNac := strings.TrimSpace(r.FormValue("lugar_nacimiento"))
	var ln sql.NullString
	if lugarNac != "" {
		ln = sql.NullString{String: lugarNac, Valid: true}
	}

	fechaNac := strings.TrimSpace(r.FormValue("fecha_nacimiento"))
	var fn sql.NullString
	if fechaNac != "" {
		fn = sql.NullString{String: fechaNac, Valid: true}
	}

	nac := strings.TrimSpace(r.FormValue("nacionalidad"))
	var n sql.NullString
	if nac != "" {
		n = sql.NullString{String: nac, Valid: true}
	}

	estCivil := strings.TrimSpace(r.FormValue("estado_civil"))
	var ec sql.NullString
	if estCivil != "" {
		ec = sql.NullString{String: estCivil, Valid: true}
	}

	tipoDoc := strings.TrimSpace(r.FormValue("tipo_documento"))
	var td sql.NullString
	if tipoDoc != "" {
		td = sql.NullString{String: tipoDoc, Valid: true}
	}

	prof := strings.TrimSpace(r.FormValue("profesion"))
	var p sql.NullString
	if prof != "" {
		p = sql.NullString{String: prof, Valid: true}
	}

	lugarTrab := strings.TrimSpace(r.FormValue("lugar_trabajo"))
	var lt sql.NullString
	if lugarTrab != "" {
		lt = sql.NullString{String: lugarTrab, Valid: true}
	}

	estado := r.FormValue("estado")
	if estado == "" {
		estado = "Pendiente"
	}

	activoVal := 1
	if r.FormValue("activo") == "0" {
		activoVal = 0
	}

	clasificacion := strings.TrimSpace(r.FormValue("clasificacion"))
	if clasificacion == "" {
		clasificacion = "Titular"
	}

	var titularID sql.NullInt64
	if clasificacion == "Adherente" {
		tIDStr := r.FormValue("titular_id")
		if tID, err := strconv.ParseInt(tIDStr, 10, 64); err == nil && tID > 0 {
			titularID = sql.NullInt64{Int64: tID, Valid: true}
		}
	}

	return sqlc.CreateSocioParams{
		Nombre:          strings.TrimSpace(r.FormValue("nombre")),
		Apellido:        strings.TrimSpace(r.FormValue("apellido")),
		LugarNacimiento: ln,
		FechaNacimiento: fn,
		Nacionalidad:    n,
		EstadoCivil:     ec,
		TipoDocumento:   td,
		NroDocumento:    strings.TrimSpace(r.FormValue("nro_documento")),
		Profesion:       p,
		LugarTrabajo:    lt,
		Domicilio:       dom,
		Telefono:        tel,
		Email:           email,
		Estado:          estado,
		Activo:          int64(activoVal),
		NumeroSocio:     num,
		Clasificacion:   clasificacion,
		TitularID:       titularID,
	}
}

func (h *SociosHandler) generateNextNumeroSocio(ctx context.Context) string {
	last, err := h.queries.GetLastNumeroSocio(ctx)
	if err != nil || !last.Valid || last.String == "" {
		return "1"
	}

	lastNum, err := strconv.Atoi(last.String)
	if err != nil {
		return "1"
	}

	return strconv.Itoa(lastNum + 1)
}
