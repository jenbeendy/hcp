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

type ScorecardHole struct {
	Index      int
	Par        int
	Length     int
	HcpIndex   int
	Strokes    string
	HcpStrokes int
	Stableford int
	ScoreClass string
}

type ScorecardGroup struct {
	Label           string
	Holes           []ScorecardHole
	LengthTotal     int
	ParTotal        int
	StrokesTotal    int
	StablefordTotal int
}

type Scorecard struct {
	GolferName      string
	CourseName      string
	StipRoundName   string
	TeeColor        string
	Date            string
	Par             int
	CR              string
	SR              int
	PCC             int
	HcpBefore       string
	HcpAfter        string
	PlayingHcp      int
	RoundIndex      int
	MultiRound      bool
	Groups          []ScorecardGroup
	Strokes         int
	StablefordNetto int
}

type ScorecardPage struct {
	TournamentID int64
	Rounds       []Scorecard
}

func RoundDetail(database *sql.DB, client *api.Client) http.HandlerFunc {
	tmpl := template.Must(template.ParseFiles("templates/scorecard.html"))
	return func(w http.ResponseWriter, r *http.Request) {
		golferID := r.PathValue("id")
		tournamentID := r.PathValue("tid")
		if _, err := strconv.Atoi(golferID); err != nil {
			http.Error(w, "Invalid golfer ID", 400)
			return
		}
		tid, err := strconv.ParseInt(tournamentID, 10, 64)
		if err != nil {
			http.Error(w, "Invalid tournament ID", 400)
			return
		}

		details, err := client.GetRoundDetail(r.Context(), tournamentID, golferID)
		if err != nil {
			http.Error(w, "API error: "+err.Error(), 502)
			return
		}

		// remember the tournament so it shows up on /tournament
		if len(details) > 0 {
			db.UpsertTournament(database, tid, details[0].CourseName, details[0].RoundDate)
		}

		page := ScorecardPage{TournamentID: tid, Rounds: make([]Scorecard, len(details))}
		for i, d := range details {
			page.Rounds[i] = convertScorecard(d, len(details) > 1)
		}
		tmpl.ExecuteTemplate(w, "scorecard", page)
	}
}

func convertScorecard(d api.RoundDetail, multiRound bool) Scorecard {
	date := d.RoundDate
	if t, err := time.Parse("2006-01-02T15:04:05", d.RoundDate); err == nil {
		date = t.Format("02.01.2006")
	}

	grouping := d.HolesGrouping
	if grouping <= 0 {
		grouping = 9
	}

	var groups []ScorecardGroup
	for start := 0; start < len(d.Holes); start += grouping {
		end := start + grouping
		if end > len(d.Holes) {
			end = len(d.Holes)
		}
		g := ScorecardGroup{Label: groupLabel(start, end, len(d.Holes))}
		for _, h := range d.Holes[start:end] {
			g.Holes = append(g.Holes, ScorecardHole{
				Index:      h.HoleIndex,
				Par:        h.Par,
				Length:     h.Length,
				HcpIndex:   h.HcpIndex,
				Strokes:    h.Strokes,
				HcpStrokes: h.HcpStrokes,
				Stableford: h.StablefordNetto,
				ScoreClass: scoreClass(h),
			})
			g.LengthTotal += h.Length
			g.ParTotal += h.Par
			g.StablefordTotal += h.StablefordNetto
			if s, err := strconv.Atoi(strings.TrimSpace(h.Strokes)); err == nil {
				g.StrokesTotal += s
			}
		}
		groups = append(groups, g)
	}

	return Scorecard{
		GolferName:      d.GolferName,
		CourseName:      d.CourseName,
		StipRoundName:   d.StipRoundName,
		TeeColor:        d.TeeColorCode,
		Date:            date,
		Par:             d.Par,
		CR:              strings.Replace(fmt.Sprintf("%.1f", d.CR), ".", ",", 1),
		SR:              d.SR,
		PCC:             d.PCC,
		HcpBefore:       d.HcpBefore,
		HcpAfter:        d.HcpAfter,
		PlayingHcp:      d.PlayingHcp,
		RoundIndex:      d.RoundIndex,
		MultiRound:      multiRound,
		Groups:          groups,
		Strokes:         d.Strokes,
		StablefordNetto: d.StablefordNetto,
	}
}

func groupLabel(start, end, total int) string {
	if total == 18 {
		if start == 0 && end == 9 {
			return "OUT"
		}
		if start == 9 {
			return "IN"
		}
	}
	return "Celkem"
}

func scoreClass(h api.RoundHole) string {
	switch h.ResType {
	case "HIO", "ALBATROS", "ALBATROSS", "EAGLE":
		return "score-eagle"
	case "BIRDIE":
		return "score-birdie"
	case "PAR":
		return "score-par"
	case "BOGEY":
		return "score-bogey"
	}
	// unknown resType (DBOGEY, TBOGEY, ...) — derive from strokes vs par
	if s, err := strconv.Atoi(strings.TrimSpace(h.Strokes)); err == nil {
		switch diff := s - h.Par; {
		case diff <= -2:
			return "score-eagle"
		case diff == -1:
			return "score-birdie"
		case diff == 0:
			return "score-par"
		case diff == 1:
			return "score-bogey"
		}
	}
	return "score-double"
}
