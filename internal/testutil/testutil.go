package testutil

import (
	"context"
	"database/sql"

	"github.com/sehogas/socios/db/sqlc"
	"github.com/sehogas/socios/internal/database"
	_ "modernc.org/sqlite"
)

// SetupTestDB inicializa una base de datos SQLite en memoria, ejecuta las migraciones
// y devuelve la conexión y el objeto Queries para los tests.
func SetupTestDB() (*sql.DB, *sqlc.Queries, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, nil, err
	}

	// Habilitar llaves foráneas
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		db.Close()
		return nil, nil, err
	}

	// Correr migraciones
	ctx := context.Background()
	err = database.RunMigrations(ctx, db)
	if err != nil {
		db.Close()
		return nil, nil, err
	}

	queries := sqlc.New(db)
	return db, queries, nil
}
