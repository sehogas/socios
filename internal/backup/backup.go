package backup

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CreateBackup realiza una copia de seguridad consistente de la base de datos SQLite
// usando el comando VACUUM INTO en la carpeta ./backups/ y devuelve la ruta del archivo generado.
func CreateBackup(ctx context.Context, db *sql.DB) (string, error) {
	backupDir := "./backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("error al crear directorio de backups: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("backup_%s.db", timestamp))

	// SQLite requiere que el archivo de destino NO exista.
	// Si existe por alguna coincidencia rara, lo borramos.
	_ = os.Remove(backupPath)

	// Ejecutar VACUUM INTO que crea un archivo db consistente y limpio
	query := fmt.Sprintf("VACUUM INTO '%s';", backupPath)
	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("error ejecutando VACUUM INTO: %w", err)
	}

	return backupPath, nil
}
