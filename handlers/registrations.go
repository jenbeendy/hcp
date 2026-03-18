package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/hcp/golf/api"
	"github.com/hcp/golf/db"
)

type DisplayRegistration struct {
	Name           string
	DateFrom       string
	DateTo         string
	CourseName     string
	NumberOfRounds int
	State          string
	Fees           string
}

type RegistrationsData struct {
	GolferID      string
	GolferName    string
	DateFrom      string
	DateTo        string
	PrevFrom      string
	PrevTo        string
	NextFrom      string
	NextTo        string
	Registrations []DisplayRegistration
}

func Registrations(database *sql.DB, client *api.Client) http.HandlerFunc {
	tmpl := template.Must(template.ParseFiles("templates/base.html", "templates/registrations.html"))
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")

		id, _ := strconv.Atoi(idStr)
		golferName := idStr
		if g, err := db.GetGolfer(database, id); err == nil {
			golferName = g.Name
		}

		now := time.Now()
		from := r.URL.Query().Get("dateFrom")
		to := r.URL.Query().Get("dateTo")
		if from == "" {
			from = now.Format("2006-01-02")
		}
		if to == "" {
			to = now.AddDate(0, 0, 30).Format("2006-01-02")
		}

		fromTime, _ := time.Parse("2006-01-02", from)
		toTime, _ := time.Parse("2006-01-02", to)

		regs, err := client.GetRegistrations(r.Context(), idStr, from, to)
		if err != nil {
			http.Error(w, "API error: "+err.Error(), 502)
			return
		}

		sort.Slice(regs, func(i, j int) bool {
			return regs[i].DateActionFrom < regs[j].DateActionFrom
		})

		display := make([]DisplayRegistration, len(regs))
		for i, reg := range regs {
			display[i] = DisplayRegistration{
				Name:           truncate(reg.Name, 55),
				DateFrom:       formatDateTime(reg.DateActionFrom),
				DateTo:         formatDateTime(reg.DateActionTo),
				CourseName:     reg.CourseName,
				NumberOfRounds: reg.NumberOfRounds,
				State:          reg.State,
				Fees:           reg.Fees,
			}
		}

		tmpl.ExecuteTemplate(w, "base", RegistrationsData{
			GolferID:      idStr,
			GolferName:    golferName,
			DateFrom:      from,
			DateTo:        to,
			PrevFrom:      fromTime.AddDate(0, 0, -30).Format("2006-01-02"),
			PrevTo:        toTime.AddDate(0, 0, -30).Format("2006-01-02"),
			NextFrom:      fromTime.AddDate(0, 0, 30).Format("2006-01-02"),
			NextTo:        toTime.AddDate(0, 0, 30).Format("2006-01-02"),
			Registrations: display,
		})
	}
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

func formatDateTime(s string) string {
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t.Format("02.01.2006")
	}
	return s
}
