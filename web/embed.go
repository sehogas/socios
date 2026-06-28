package web

import "embed"

// Assets contiene todos los archivos estáticos y plantillas del frontend
//go:embed templates static
var Assets embed.FS
