package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hcp/golf/api"
	"github.com/hcp/golf/db"
)

type DisplayRecord struct {
	Date         string
	Par          int
	CR           string
	SR           int
	Strokes      string
	Points       int
	UHV          int
	PCC          int
	SU           int
	PO           string
	WhsHI        string
	RowClass       string
	TournamentID   int64
	TournamentName string
	Clickable      bool
}

type GolferData struct {
	ID           int
	FullName     string
	MemberNumber string
	HomeClub     string
	CurrentHI    string
	Records      []DisplayRecord
}

func Golfer(database *sql.DB, client *api.Client) http.HandlerFunc {
	tmpl := template.Must(template.ParseFiles("templates/base.html", "templates/golfer.html"))
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid ID", 400)
			return
		}

		// golfers linked from tournament results may not be stored
		// locally — show their HCP history anyway
		if _, err := db.GetGolfer(database, id); err != nil && err != sql.ErrNoRows {
			http.Error(w, "DB error", 500)
			return
		}

		history, err := client.GetHCPHistory(r.Context(), idStr)
		if err != nil {
			http.Error(w, "API error: "+err.Error(), 502)
			return
		}

		records := make([]DisplayRecord, len(history.HcpRecords))
		for i, rec := range history.HcpRecords {
			records[i] = convertRecord(rec)
		}

		tmpl.ExecuteTemplate(w, "base", GolferData{
			ID:           id,
			FullName:     history.FullName,
			MemberNumber: history.MemberNumber,
			HomeClub:     history.HomeClub,
			CurrentHI:    history.CurrentHI,
			Records:      records,
		})
	}
}

func convertRecord(r api.HCPRecord) DisplayRecord {
	strokes := "---"
	if r.Strokes != nil {
		strokes = strconv.Itoa(*r.Strokes)
	}

	date := r.Date
	if t, err := time.Parse("2006-01-02T15:04:05", r.Date); err == nil {
		date = t.Format("02.01.2006")
	}

	return DisplayRecord{
		Date:         date,
		Par:          r.Par,
		CR:           strings.Replace(fmt.Sprintf("%.1f", r.CR), ".", ",", 1),
		SR:           r.SR,
		Strokes:      strokes,
		Points:       r.Points,
		UHV:          r.UHV,
		PCC:          r.PCC,
		SU:           r.SU,
		PO:           strings.Replace(fmt.Sprintf("%.1f", r.PO), ".", ",", 1),
		WhsHI:        r.WhsHI,
		RowClass:       rowClass(r),
		TournamentID:   r.TournamentID,
		TournamentName: r.TournamentName,
		Clickable:      r.TournamentID != 0 && r.Strokes != nil,
	}
}

func rowClass(r api.HCPRecord) string {
	if r.OutstandingResult {
		return "hilightExtraResult"
	}
	for _, ind := range r.Indicators {
		if ind == "INPUT" {
			return "hilightAverageColor"
		}
	}
	for _, ind := range r.Indicators {
		if ind == "LAST20" {
			return "hilightInputColor"
		}
	}
	return ""
}
