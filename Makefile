.PHONY: run build sqlc docker-build docker-run docker-stop clean

# Ejecutar el servidor localmente
run:
	go run cmd/server/main.go

# Compilar el binario del servidor sin CGO
build:
	CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/socios cmd/server/main.go

# Generar el código de base de datos a través de sqlc
sqlc:
	sqlc generate

# Construir la imagen de Docker
docker-build:
	docker build -t socios .

# Construir y ejecutar el contenedor Docker mapeando volúmenes locales
docker-run:
	mkdir -p data backups
	docker run -d \
		-p 8080:8080 \
		-v $$(pwd)/data:/app/data \
		-v $$(pwd)/backups:/app/backups \
		--name club_socios \
		socios

# Detener y eliminar el contenedor de Docker
docker-stop:
	docker stop club_socios || true
	docker rm club_socios || true

# Limpiar archivos compilados
clean:
	rm -rf bin/
	rm -f database.db database.db-shm database.db-wal
	rm -rf backups/
