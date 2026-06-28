package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sehogas/socios3/internal/database"
	"github.com/sehogas/socios3/internal/server"
)

func main() {
	// Obtener puerto y ruta de base de datos desde variables de entorno o usar defaults
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./database.db"
	}

	log.Printf("Iniciando aplicación en puerto %s y base de datos en %s", port, dbPath)

	// 1. Inicializar conexión a base de datos SQLite
	db, err := database.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Error crítico al inicializar base de datos: %v", err)
	}
	defer db.Close()

	// 2. Ejecutar migraciones automáticas (schema.sql embebido)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = database.RunMigrations(ctx, db)
	if err != nil {
		log.Fatalf("Error crítico al ejecutar migraciones automáticas: %v", err)
	}
	log.Println("Migraciones ejecutadas / base de datos lista.")

	// 3. Configurar ruteo e iniciar servidor HTTP
	srvHandler := server.NewServer(db)
	
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      srvHandler,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("Servidor escuchando en http://localhost:%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error en servidor HTTP: %v", err)
	}
}
