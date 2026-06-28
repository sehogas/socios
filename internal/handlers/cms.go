package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/sehogas/socios3/db/sqlc"
	"github.com/sehogas/socios3/internal/middleware"
)

type CMSHandler struct {
	queries *sqlc.Queries
}

func NewCMSHandler(queries *sqlc.Queries) *CMSHandler {
	return &CMSHandler{queries: queries}
}

// AdminCMS muestra el listado de páginas y el formulario de creación/edición para admins
func (h *CMSHandler) AdminCMS(w http.ResponseWriter, r *http.Request) {
	paginas, err := h.queries.ListPaginas(r.Context())
	if err != nil {
		http.Error(w, "Error al listar paginas: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Paginas": paginas,
	}

	// Si se solicita editar, cargamos los datos del post
	editIDStr := r.URL.Query().Get("edit_id")
	if editIDStr != "" {
		editID, err := strconv.ParseInt(editIDStr, 10, 64)
		if err == nil {
			pagina, err := h.queries.GetPaginaById(r.Context(), editID)
			if err == nil {
				data["Pagina"] = pagina
			}
		}
	}

	RenderTemplate(w, r, "admin/cms.html", data)
}

// CreatePage registra un nuevo post en el CMS
func (h *CMSHandler) CreatePage(w http.ResponseWriter, r *http.Request) {
	session := middleware.GetSession(r.Context())
	if session == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	titulo := strings.TrimSpace(r.FormValue("titulo"))
	slug := strings.TrimSpace(strings.ToLower(r.FormValue("slug")))
	contenido := r.FormValue("contenido")
	estado := r.FormValue("estado") // Borrador, Publicado
	visibilidad := r.FormValue("visibilidad") // publico, socios, admin

	if titulo == "" || slug == "" || contenido == "" {
		http.Redirect(w, r, "/admin/paginas?error=Completa todos los campos obligatorios", http.StatusSeeOther)
		return
	}

	_, err := h.queries.CreatePagina(r.Context(), sqlc.CreatePaginaParams{
		Titulo:      titulo,
		Slug:        slug,
		Contenido:   contenido,
		Estado:      estado,
		Visibilidad: visibilidad,
		AutorID:     session.UserID,
	})
	if err != nil {
		http.Redirect(w, r, "/admin/paginas?error=Error al crear la pagina (el slug debe ser unico): "+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/paginas?success=Pagina creada correctamente", http.StatusSeeOther)
}

// UpdatePage guarda los cambios en una página existente
func (h *CMSHandler) UpdatePage(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/paginas?error=ID invalido", http.StatusSeeOther)
		return
	}

	titulo := strings.TrimSpace(r.FormValue("titulo"))
	slug := strings.TrimSpace(strings.ToLower(r.FormValue("slug")))
	contenido := r.FormValue("contenido")
	estado := r.FormValue("estado")
	visibilidad := r.FormValue("visibilidad")

	if titulo == "" || slug == "" || contenido == "" {
		http.Redirect(w, r, fmt.Sprintf("/admin/paginas?edit_id=%d&error=Campos obligatorios vacios", id), http.StatusSeeOther)
		return
	}

	err = h.queries.UpdatePagina(r.Context(), sqlc.UpdatePaginaParams{
		Titulo:      titulo,
		Slug:        slug,
		Contenido:   contenido,
		Estado:      estado,
		Visibilidad: visibilidad,
		ID:          id,
	})
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/paginas?edit_id=%d&error=Error al guardar: %s", id, err.Error()), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/paginas?success=Pagina actualizada con exito", http.StatusSeeOther)
}

// DeletePage elimina un post
func (h *CMSHandler) DeletePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/paginas?error=ID invalido", http.StatusSeeOther)
		return
	}

	err = h.queries.DeletePagina(r.Context(), id)
	if err != nil {
		http.Redirect(w, r, "/admin/paginas?error=Error al eliminar pagina", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/paginas?success=Pagina eliminada correctamente", http.StatusSeeOther)
}

// ViewCMS gestiona la vista pública de novedades (lista y detalle)
func (h *CMSHandler) ViewCMS(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Query().Get("slug")
	session := middleware.GetSession(r.Context())

	// Determinar el rol/estado del visitante
	isSocioActivo := false
	isAdmin := false
	if session != nil {
		if session.Rol == "admin" {
			isAdmin = true
		} else {
			// Comprobar si el user tiene una ficha de socio activa en la DB
			socio, err := h.queries.GetSocioByEmail(r.Context(), sql.NullString{String: session.Email, Valid: true})
			if err == nil && socio.Activo == 1 {
				isSocioActivo = true
			}
		}
	}

	data := make(map[string]interface{})

	if slug != "" {
		// Mostrar artículo individual
		pagina, err := h.queries.GetPaginaBySlug(r.Context(), slug)
		if err != nil || pagina.Estado == "Borrador" && !isAdmin {
			http.Error(w, "Pagina no encontrada", http.StatusNotFound)
			return
		}

		// Validar Visibilidad / Autorización
		switch pagina.Visibilidad {
		case "socios":
			if !isSocioActivo && !isAdmin {
				http.Error(w, "Acceso denegado: este contenido es exclusivo para socios activos.", http.StatusForbidden)
				return
			}
		case "admin":
			if !isAdmin {
				http.Error(w, "Acceso denegado: contenido reservado para administradores.", http.StatusForbidden)
				return
			}
		}

		data["Pagina"] = pagina
		data["ContenidoHTML"] = template.HTML(pagina.Contenido) // Permitir HTML seguro en CMS
		RenderTemplate(w, r, "cms/view.html", data)

	} else {
		// Mostrar listado de artículos permitidos
		todas, err := h.queries.ListPaginas(r.Context())
		if err != nil {
			http.Error(w, "Error al listar novedades", http.StatusInternalServerError)
			return
		}

		var filtradas []sqlc.Pagina
		for _, p := range todas {
			// Ignorar borradores para usuarios normales
			if p.Estado == "Borrador" && !isAdmin {
				continue
			}

			// Filtrar según visibilidad
			if p.Visibilidad == "publico" {
				filtradas = append(filtradas, p)
			} else if p.Visibilidad == "socios" && (isSocioActivo || isAdmin) {
				filtradas = append(filtradas, p)
			} else if p.Visibilidad == "admin" && isAdmin {
				filtradas = append(filtradas, p)
			}
		}

		data["Paginas"] = filtradas
		RenderTemplate(w, r, "cms/view.html", data)
	}
}
