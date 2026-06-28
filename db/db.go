package db

import _ "embed"

// SchemaSQL contiene las instrucciones DDL de inicialización de la base de datos.
//go:embed schema.sql
var SchemaSQL string
