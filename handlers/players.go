package handlers

import (
	"database/sql"
	"html/template"
	"net/http"

	"github.com/hcp/golf/db"
)

func Players(database *sql.DB) http.HandlerFunc {
	tmpl := template.Must(template.ParseFiles("templates/base.html", "templates/players.html"))
	return func(w http.ResponseWriter, r *http.Request) {
		golfers, err := db.ListGolfers(database)
		if err != nil {
			http.Error(w, "DB error", 500)
			return
		}
		tmpl.ExecuteTemplate(w, "base", golfers)
	}
}
