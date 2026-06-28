# Stage 1: Compilación del binario estático en Go
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Descargar dependencias
COPY go.mod ./
# Si existe go.sum lo copiamos
COPY go.sum* ./
RUN go mod download

# Copiar el código fuente
COPY . .

# Argumento de compilación para definir la versión
ARG VERSION=dev

# Compilar binario deshabilitando CGO (gracias al driver de SQLite puro)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s -X main.Version=${VERSION}" -o /app/socios cmd/server/main.go

# Stage 2: Contenedor liviano de ejecución
FROM alpine:3.19

WORKDIR /app

# Copiar el binario compilado
COPY --from=builder /app/socios /app/socios

# Crear directorios para persistencia de base de datos y backups
RUN mkdir -p /app/data /app/backups

# Exponer puerto por defecto
EXPOSE 8080

# Variables de entorno por defecto para el contenedor
ENV PORT=8080
ENV DATABASE_PATH=/app/data/database.db

# Definir volúmenes para asegurar la persistencia de datos y copias de seguridad
VOLUME ["/app/data", "/app/backups"]

# Comando de inicio
CMD ["/app/socios"]
