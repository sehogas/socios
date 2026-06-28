package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/sehogas/socios/db/sqlc"
	"github.com/sehogas/socios/internal/middleware"
	"github.com/sehogas/socios/web"
)

var argLocation *time.Location
var dbInstance *sql.DB
var queriesInstance *sqlc.Queries

func SetDatabase(db *sql.DB, q *sqlc.Queries) {
	dbInstance = db
	queriesInstance = q
}

func init() {
	var err error
	argLocation, err = time.LoadLocation("America/Argentina/Buenos_Aires")
	if err != nil {
		// Fallback robusto en caso de que el entorno (como Docker) carezca de la base tzdata
		argLocation = time.FixedZone("ART", -3*60*60)
	}
}

// RenderTemplate parsea y renderiza una plantilla de Go HTML embebida,
// inyectando automáticamente la sesión activa y los mensajes de éxito/error.
func RenderTemplate(w http.ResponseWriter, r *http.Request, name string, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}

	// Inyectar nombre del sistema
	nombreSistema := "Club de Socios"
	if queriesInstance != nil {
		if val, err := queriesInstance.GetConfig(r.Context(), "nombre_sistema"); err == nil && val != "" {
			nombreSistema = val
		}
	}
	data["NombreSistema"] = nombreSistema

	// Inyectar datos de sesión si existen en el contexto
	if sess := middleware.GetSession(r.Context()); sess != nil {
		data["Session"] = sess
	}

	// Inyectar alertas pasadas por query strings
	if success := r.URL.Query().Get("success"); success != "" {
		data["SuccessMsg"] = success
	}
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		data["ErrorMsg"] = errMsg
	}

	// Definir funciones auxiliares para las plantillas HTML
	funcMap := template.FuncMap{
		"summary": func(s string) string {
			stripped := stripHTML(s)
			if len(stripped) <= 150 {
				return stripped
			}
			return stripped[:150] + "..."
		},
		"formatDate": func(t time.Time) string {
			return t.In(argLocation).Format("02/01/2006 15:04")
		},
	}

	// Parsear layout base y la plantilla objetivo cargando las funciones auxiliares
	tmpl := template.New("base").Funcs(funcMap)
	tmpl, err := tmpl.ParseFS(web.Assets, "templates/layout.html", "templates/"+name)
	if err != nil {
		http.Error(w, "Error de sistema al cargar plantillas: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = tmpl.ExecuteTemplate(w, "layout", data)
	if err != nil {
		http.Error(w, "Error de sistema al renderizar página: "+err.Error(), http.StatusInternalServerError)
	}
}

// stripHTML remueve todas las etiquetas HTML de un string de forma simple y eficiente
func stripHTML(s string) string {
	var builder strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			builder.WriteRune(r)
		}
	}
	return strings.TrimSpace(builder.String())
}
