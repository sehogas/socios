CREATE TABLE usuarios (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    rol TEXT NOT NULL CHECK(rol IN ('admin', 'user')),
    activo INTEGER DEFAULT 1 CHECK(activo IN (0, 1)),
    verificado INTEGER DEFAULT 0 CHECK(verificado IN (0, 1)) NOT NULL,
    fecha_creacion TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    usuario_id INTEGER NOT NULL,
    token TEXT UNIQUE NOT NULL,
    tipo TEXT NOT NULL CHECK(tipo IN ('VERIFICACION_EMAIL', 'RECUPERACION_PASSWORD')),
    expiracion TIMESTAMP NOT NULL,
    fecha_creacion TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY(usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE
);

CREATE TABLE socios (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    numero_socio TEXT UNIQUE,
    nombre TEXT NOT NULL,
    apellido TEXT NOT NULL,
    lugar_nacimiento TEXT,
    fecha_nacimiento TEXT, -- YYYY-MM-DD
    nacionalidad TEXT,
    estado_civil TEXT,
    tipo_documento TEXT,
    nro_documento TEXT UNIQUE NOT NULL,
    profesion TEXT,
    lugar_trabajo TEXT,
    domicilio TEXT,
    telefono TEXT,
    email TEXT UNIQUE,
    estado TEXT DEFAULT 'Pendiente' CHECK(estado IN ('Pendiente', 'Aprobado', 'Rechazado')) NOT NULL,
    activo INTEGER DEFAULT 1 CHECK(activo IN (0, 1)) NOT NULL,
    fecha_aprobacion TEXT, -- YYYY-MM-DD
    clasificacion TEXT NOT NULL DEFAULT 'Titular' CHECK(clasificacion IN ('Titular', 'Adherente', 'Honorario', 'Vitalicio', 'Temporario')),
    titular_id INTEGER REFERENCES socios(id),
    fecha_creacion TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE paginas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    titulo TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    contenido TEXT NOT NULL,
    estado TEXT NOT NULL CHECK(estado IN ('Borrador', 'Publicado')),
    visibilidad TEXT NOT NULL CHECK(visibilidad IN ('publico', 'socios', 'admin')),
    autor_id INTEGER NOT NULL,
    fecha_actualizacion TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY(autor_id) REFERENCES usuarios(id)
);

CREATE TABLE cuotas_config (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    categoria TEXT UNIQUE NOT NULL,
    monto REAL NOT NULL
);

CREATE TABLE cuotas_generadas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    socio_id INTEGER NOT NULL,
    periodo TEXT NOT NULL, -- YYYY-MM
    monto_original REAL NOT NULL,
    monto_pendiente REAL NOT NULL,
    estado TEXT DEFAULT 'Impaga' CHECK(estado IN ('Impaga', 'Parcial', 'Paga')) NOT NULL,
    FOREIGN KEY(socio_id) REFERENCES socios(id)
);

CREATE TABLE transacciones_cta_cte (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    socio_id INTEGER NOT NULL,
    tipo TEXT NOT NULL CHECK(tipo IN ('DEBITO', 'CREDITO')),
    monto REAL NOT NULL,
    fecha TEXT NOT NULL, -- YYYY-MM-DD
    descripcion TEXT,
    transaccion_caja_id INTEGER,
    FOREIGN KEY(socio_id) REFERENCES socios(id),
    FOREIGN KEY(transaccion_caja_id) REFERENCES transacciones_caja(id)
);

CREATE TABLE transacciones_caja (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tipo TEXT NOT NULL CHECK(tipo IN ('INGRESO', 'EGRESO')),
    cuenta TEXT NOT NULL, -- 'Efectivo', 'Banco', 'MercadoPago'
    monto REAL NOT NULL,
    fecha TEXT NOT NULL, -- YYYY-MM-DD
    categoria TEXT NOT NULL, -- 'Servicio', 'Sueldo', 'Cobro Cuota', 'Adelanto', 'Alquiler'
    descripcion TEXT,
    fecha_creacion TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE config (
    clave TEXT PRIMARY KEY,
    valor TEXT NOT NULL
);

CREATE TABLE valores_cuota (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    clasificacion TEXT NOT NULL CHECK(clasificacion IN ('Titular', 'Adherente', 'Honorario', 'Vitalicio', 'Temporario')),
    monto REAL NOT NULL,
    vigencia_inicial TEXT NOT NULL, -- YYYY-MM
    fecha_creacion TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    UNIQUE(clasificacion, vigencia_inicial)
);
