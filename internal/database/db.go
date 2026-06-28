package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// InitDB abre la base de datos SQLite habilitando llaves foráneas y modo WAL
func InitDB(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "/" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("error al crear directorio para base de datos: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error al abrir base de datos: %w", err)
	}

	// Habilitar llaves foráneas y modo WAL para mejor concurrencia en SQLite
	_, err = db.Exec("PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("error al configurar PRAGMAs de SQLite: %w", err)
	}

	return db, nil
}
