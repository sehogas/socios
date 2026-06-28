-- Usuarios Queries
-- name: CreateUser :one
INSERT INTO usuarios (email, password_hash, rol, activo, verificado)
VALUES (?, ?, ?, ?, ?)
RETURNING id, email, password_hash, rol, activo, verificado, fecha_creacion;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, rol, activo, verificado, fecha_creacion FROM usuarios WHERE email = ? LIMIT 1;

-- name: GetUserById :one
SELECT id, email, password_hash, rol, activo, verificado, fecha_creacion FROM usuarios WHERE id = ? LIMIT 1;

-- name: ListUsers :many
SELECT id, email, password_hash, rol, activo, verificado, fecha_creacion FROM usuarios ORDER BY fecha_creacion DESC;

-- name: UpdateUserStatus :exec
UPDATE usuarios SET activo = ? WHERE id = ?;

-- name: UpdateUserRole :exec
UPDATE usuarios SET rol = ? WHERE id = ?;

-- name: UpdateUserVerification :exec
UPDATE usuarios SET verificado = ? WHERE id = ?;

-- name: UpdateUserPassword :exec
UPDATE usuarios SET password_hash = ? WHERE id = ?;


-- Tokens Queries
-- name: CreateToken :one
INSERT INTO tokens (usuario_id, token, tipo, expiracion)
VALUES (?, ?, ?, ?)
RETURNING id, usuario_id, token, tipo, expiracion, fecha_creacion;

-- name: GetTokenByValueAndTipo :one
SELECT id, usuario_id, token, tipo, expiracion, fecha_creacion
FROM tokens
WHERE token = ? AND tipo = ? LIMIT 1;

-- name: DeleteToken :exec
DELETE FROM tokens WHERE id = ?;

-- name: DeleteTokensByUsuarioAndTipo :exec
DELETE FROM tokens WHERE usuario_id = ? AND tipo = ?;


-- Socios Queries
-- name: CreateSocio :one
INSERT INTO socios (
    nombre, apellido, lugar_nacimiento, fecha_nacimiento, nacionalidad,
    estado_civil, tipo_documento, nro_documento, profesion, lugar_trabajo,
    domicilio, telefono, email, estado, activo, numero_socio, fecha_aprobacion,
    clasificacion, titular_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, numero_socio, nombre, apellido, lugar_nacimiento, fecha_nacimiento, nacionalidad, estado_civil, tipo_documento, nro_documento, profesion, lugar_trabajo, domicilio, telefono, email, estado, activo, fecha_aprobacion, clasificacion, titular_id, fecha_creacion;

-- name: GetSocioById :one
SELECT id, numero_socio, nombre, apellido, lugar_nacimiento, fecha_nacimiento, nacionalidad, estado_civil, tipo_documento, nro_documento, profesion, lugar_trabajo, domicilio, telefono, email, estado, activo, fecha_aprobacion, clasificacion, titular_id, fecha_creacion FROM socios WHERE id = ? LIMIT 1;

-- name: GetSocioByEmail :one
SELECT id, numero_socio, nombre, apellido, lugar_nacimiento, fecha_nacimiento, nacionalidad, estado_civil, tipo_documento, nro_documento, profesion, lugar_trabajo, domicilio, telefono, email, estado, activo, fecha_aprobacion, clasificacion, titular_id, fecha_creacion FROM socios WHERE email = ? LIMIT 1;

-- name: GetSocioByDocumento :one
SELECT id, numero_socio, nombre, apellido, lugar_nacimiento, fecha_nacimiento, nacionalidad, estado_civil, tipo_documento, nro_documento, profesion, lugar_trabajo, domicilio, telefono, email, estado, activo, fecha_aprobacion, clasificacion, titular_id, fecha_creacion FROM socios WHERE nro_documento = ? LIMIT 1;

-- name: ListSocios :many
SELECT id, numero_socio, nombre, apellido, lugar_nacimiento, fecha_nacimiento, nacionalidad, estado_civil, tipo_documento, nro_documento, profesion, lugar_trabajo, domicilio, telefono, email, estado, activo, fecha_aprobacion, clasificacion, titular_id, fecha_creacion FROM socios ORDER BY id DESC;

-- name: GetLastNumeroSocio :one
SELECT numero_socio FROM socios 
WHERE numero_socio IS NOT NULL AND numero_socio != ''
ORDER BY CAST(numero_socio AS INTEGER) DESC LIMIT 1;

-- name: UpdateSocioStatus :exec
UPDATE socios 
SET estado = ?, numero_socio = ?, fecha_aprobacion = ? 
WHERE id = ?;

-- name: UpdateSocioActive :exec
UPDATE socios SET activo = ? WHERE id = ?;

-- name: UpdateSocio :exec
UPDATE socios
SET nombre = ?, apellido = ?, lugar_nacimiento = ?, fecha_nacimiento = ?, 
    nacionalidad = ?, estado_civil = ?, tipo_documento = ?, nro_documento = ?, 
    profesion = ?, lugar_trabajo = ?, domicilio = ?, telefono = ?, email = ?,
    clasificacion = ?, titular_id = ?
WHERE id = ?;



-- CMS / Paginas Queries
-- name: CreatePagina :one
INSERT INTO paginas (titulo, slug, contenido, estado, visibilidad, autor_id)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id, titulo, slug, contenido, estado, visibilidad, autor_id, fecha_actualizacion;

-- name: GetPaginaBySlug :one
SELECT id, titulo, slug, contenido, estado, visibilidad, autor_id, fecha_actualizacion FROM paginas WHERE slug = ? LIMIT 1;

-- name: GetPaginaById :one
SELECT id, titulo, slug, contenido, estado, visibilidad, autor_id, fecha_actualizacion FROM paginas WHERE id = ? LIMIT 1;

-- name: ListPaginas :many
SELECT id, titulo, slug, contenido, estado, visibilidad, autor_id, fecha_actualizacion FROM paginas ORDER BY fecha_actualizacion DESC;

-- name: UpdatePagina :exec
UPDATE paginas
SET titulo = ?, slug = ?, contenido = ?, estado = ?, visibilidad = ?, fecha_actualizacion = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeletePagina :exec
DELETE FROM paginas WHERE id = ?;


-- Cuotas Config Queries
-- name: CreateOrUpdateCuotaConfig :one
INSERT INTO cuotas_config (categoria, monto)
VALUES (?, ?)
ON CONFLICT(categoria) DO UPDATE SET monto = excluded.monto
RETURNING id, categoria, monto;

-- name: GetCuotaConfigByCategoria :one
SELECT id, categoria, monto FROM cuotas_config WHERE categoria = ? LIMIT 1;

-- name: ListCuotasConfig :many
SELECT id, categoria, monto FROM cuotas_config;

-- Config Queries
-- name: GetConfig :one
SELECT valor FROM config WHERE clave = ? LIMIT 1;

-- name: SetConfig :exec
INSERT INTO config (clave, valor) VALUES (?, ?)
ON CONFLICT(clave) DO UPDATE SET valor = excluded.valor;

-- Valores Cuota Queries
-- name: GetCuotaValorByClasificacionAndPeriodo :one
SELECT id, clasificacion, monto, vigencia_inicial, fecha_creacion
FROM valores_cuota
WHERE clasificacion = ? AND vigencia_inicial <= ?
ORDER BY vigencia_inicial DESC LIMIT 1;

-- name: CreateOrUpdateCuotaValor :one
INSERT INTO valores_cuota (clasificacion, monto, vigencia_inicial)
VALUES (?, ?, ?)
ON CONFLICT(clasificacion, vigencia_inicial) DO UPDATE SET monto = excluded.monto
RETURNING id, clasificacion, monto, vigencia_inicial, fecha_creacion;

-- name: ListCuotasValores :many
SELECT id, clasificacion, monto, vigencia_inicial, fecha_creacion
FROM valores_cuota
ORDER BY clasificacion ASC, vigencia_inicial DESC;



-- Cuotas Generadas Queries
-- name: CreateCuotaGenerada :one
INSERT INTO cuotas_generadas (socio_id, periodo, monto_original, monto_pendiente, estado)
VALUES (?, ?, ?, ?, ?)
RETURNING id, socio_id, periodo, monto_original, monto_pendiente, estado;

-- name: GetCuotaGeneradaBySocioAndPeriodo :one
SELECT id, socio_id, periodo, monto_original, monto_pendiente, estado FROM cuotas_generadas WHERE socio_id = ? AND periodo = ? LIMIT 1;

-- name: ListCuotasGeneradasBySocio :many
SELECT id, socio_id, periodo, monto_original, monto_pendiente, estado FROM cuotas_generadas WHERE socio_id = ? ORDER BY periodo DESC;

-- name: UpdateCuotaGeneradaMontoPendiente :exec
UPDATE cuotas_generadas 
SET monto_pendiente = ?, estado = ? 
WHERE id = ?;


-- Cuenta Corriente Queries
-- name: CreateTransaccionCtaCte :one
INSERT INTO transacciones_cta_cte (socio_id, tipo, monto, fecha, descripcion, transaccion_caja_id)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id, socio_id, tipo, monto, fecha, descripcion, transaccion_caja_id;

-- name: ListTransaccionesCtaCteBySocio :many
SELECT id, socio_id, tipo, monto, fecha, descripcion, transaccion_caja_id FROM transacciones_cta_cte WHERE socio_id = ? ORDER BY fecha DESC, id DESC;

-- name: GetSaldoSocio :one
SELECT CAST(COALESCE(SUM(CASE WHEN tipo = 'CREDITO' THEN monto ELSE 0 END) - 
                     SUM(CASE WHEN tipo = 'DEBITO' THEN monto ELSE 0 END), 0.0) AS REAL) AS saldo
FROM transacciones_cta_cte
WHERE socio_id = ?;


-- Caja / Tesoreria Queries
-- name: CreateTransaccionCaja :one
INSERT INTO transacciones_caja (tipo, cuenta, monto, fecha, categoria, descripcion)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id, tipo, cuenta, monto, fecha, categoria, descripcion, fecha_creacion;

-- name: ListTransaccionesCaja :many
SELECT id, tipo, cuenta, monto, fecha, categoria, descripcion, fecha_creacion
FROM transacciones_caja
ORDER BY fecha DESC, id DESC LIMIT 1000000;

-- name: GetTotalIngresosCaja :one
SELECT CAST(COALESCE(SUM(monto), 0.0) AS REAL) FROM transacciones_caja WHERE tipo = 'INGRESO';

-- name: GetTotalEgresosCaja :one
SELECT CAST(COALESCE(SUM(monto), 0.0) AS REAL) FROM transacciones_caja WHERE tipo = 'EGRESO';

-- name: GetCajaBalanceByCuenta :one
SELECT CAST(COALESCE(SUM(CASE WHEN tipo = 'INGRESO' THEN monto ELSE -monto END), 0.0) AS REAL)
FROM transacciones_caja
WHERE cuenta = ?;

-- name: GetCajaSummaryByCategory :many
SELECT tipo, categoria, CAST(SUM(monto) AS REAL) as total
FROM transacciones_caja
GROUP BY tipo, categoria
ORDER BY tipo ASC, total DESC;
