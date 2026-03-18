package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"unicode"

	"github.com/hcp/golf/db"
)

type displayGolfer struct {
	ID      int
	Name    string
	surname string // for sorting only
}

// All-caps words are treated as the surname.
func formatName(raw string) (display, surname string) {
	parts := strings.Fields(raw)
	var surnParts, givenParts []string
	for _, p := range parts {
		if isAllUpper(p) {
			surnParts = append(surnParts, titleCase(p))
		} else {
			givenParts = append(givenParts, p)
		}
	}
	if len(surnParts) == 0 {
		return raw, strings.ToLower(raw)
	}
	display = strings.Join(append(surnParts, givenParts...), " ")
	surname = strings.ToLower(strings.Join(surnParts, " "))
	return
}

func isAllUpper(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) && !unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

func titleCase(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return string(unicode.ToUpper(r[0])) + strings.ToLower(string(r[1:]))
}

func Players(database *sql.DB) http.HandlerFunc {
	tmpl := template.Must(template.ParseFiles("templates/base.html", "templates/players.html"))
	return func(w http.ResponseWriter, r *http.Request) {
		golfers, err := db.ListGolfers(database)
		if err != nil {
			http.Error(w, "DB error", 500)
			return
		}

		display := make([]displayGolfer, len(golfers))
		for i, g := range golfers {
			name, surn := formatName(g.Name)
			display[i] = displayGolfer{ID: g.ID, Name: name, surname: surn}
		}
		sort.Slice(display, func(i, j int) bool {
			return display[i].surname < display[j].surname
		})

		tmpl.ExecuteTemplate(w, "base", display)
	}
}
