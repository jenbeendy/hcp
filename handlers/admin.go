package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"net/url"
	"strconv"

	"github.com/hcp/golf/api"
	"github.com/hcp/golf/db"
)

type AdminData struct {
	Error   string
	Success string
}

func AdminGet(database *sql.DB) http.HandlerFunc {
	tmpl := template.Must(template.ParseFiles("templates/base.html", "templates/admin.html"))
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "base", AdminData{
			Error:   r.URL.Query().Get("error"),
			Success: r.URL.Query().Get("success"),
		})
	}
}

func AdminPost(database *sql.DB, client *api.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.FormValue("golfer_id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Redirect(w, r, "/adminpage?error="+url.QueryEscape("Neplatné ID"), http.StatusSeeOther)
			return
		}

		exists, err := db.GolferExists(database, id)
		if err != nil {
			http.Redirect(w, r, "/adminpage?error="+url.QueryEscape("Chyba databáze"), http.StatusSeeOther)
			return
		}
		if exists {
			http.Redirect(w, r, "/adminpage?error="+url.QueryEscape("Golfista již existuje"), http.StatusSeeOther)
			return
		}

		history, err := client.GetHCPHistory(r.Context(), idStr)
		if err != nil {
			http.Redirect(w, r, "/adminpage?error="+url.QueryEscape("Golfista nenalezen v CGF"), http.StatusSeeOther)
			return
		}

		if err := db.AddGolfer(database, id, history.FullName); err != nil {
			http.Redirect(w, r, "/adminpage?error="+url.QueryEscape("Chyba při uložení"), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/adminpage?success="+url.QueryEscape("Přidán: "+history.FullName), http.StatusSeeOther)
	}
}
