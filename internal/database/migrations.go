package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sehogas/socios/db"
)

// RunMigrations lee el esquema SQL embebido expuesto por el paquete db y lo ejecuta
// en la base de datos si detecta que la tabla principal 'usuarios' no existe.
func RunMigrations(ctx context.Context, database *sql.DB) error {
	var name string
	err := database.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='usuarios';").Scan(&name)
	if err == sql.ErrNoRows {
		// La base de datos no está inicializada. Ejecutamos el DDL completo.
		_, err = database.ExecContext(ctx, db.SchemaSQL)
		if err != nil {
			return fmt.Errorf("error al ejecutar DDL del esquema: %w", err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("error al verificar inicialización de la base de datos: %w", err)
	}

	// La base de datos ya está inicializada, pero puede faltar clasificacion, titular_id, config o valores_cuota.
	var hasClasificacion int
	err = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('socios') WHERE name='clasificacion';").Scan(&hasClasificacion)
	if err == nil && hasClasificacion == 0 {
		_, err = database.ExecContext(ctx, "ALTER TABLE socios ADD COLUMN clasificacion TEXT NOT NULL DEFAULT 'Titular';")
		if err != nil {
			return fmt.Errorf("error al agregar columna clasificacion: %w", err)
		}
	}

	var hasTitularId int
	err = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('socios') WHERE name='titular_id';").Scan(&hasTitularId)
	if err == nil && hasTitularId == 0 {
		_, err = database.ExecContext(ctx, "ALTER TABLE socios ADD COLUMN titular_id INTEGER REFERENCES socios(id);")
		if err != nil {
			return fmt.Errorf("error al agregar columna titular_id: %w", err)
		}
	}

	_, err = database.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS config (
			clave TEXT PRIMARY KEY,
			valor TEXT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("error al crear tabla config: %w", err)
	}

	_, err = database.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS valores_cuota (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			clasificacion TEXT NOT NULL CHECK(clasificacion IN ('Titular', 'Adherente', 'Honorario', 'Vitalicio', 'Temporario')),
			monto REAL NOT NULL,
			vigencia_inicial TEXT NOT NULL, -- YYYY-MM
			fecha_creacion TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
			UNIQUE(clasificacion, vigencia_inicial)
		);
	`)
	if err != nil {
		return fmt.Errorf("error al crear tabla valores_cuota: %w", err)
	}

	// Insertar nombre del sistema por defecto
	_, err = database.ExecContext(ctx, "INSERT OR IGNORE INTO config (clave, valor) VALUES ('nombre_sistema', 'Club de Socios');")
	if err != nil {
		return fmt.Errorf("error al inicializar nombre_sistema por defecto: %w", err)
	}

	return nil
}
