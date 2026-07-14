package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hcp/golf/api"
	"github.com/hcp/golf/db"
)

type TournamentListItem struct {
	ID         int64
	CourseName string
	Date       string
}

type TournamentListData struct {
	Tournaments []TournamentListItem
}

type CategoryOption struct {
	ID       int64
	Name     string
	Selected bool
}

type TournamentResultsData struct {
	TournamentID int64
	Title        string
	Categories   []CategoryOption
	SelectedID   int64
}

type ResultRow struct {
	Place      string
	Name       string
	Club       string
	HCP        string
	Strokes    string
	Stableford string
	Pending    bool
	NoResult   bool
}

type ResultRowsData struct {
	Rows       []ResultRow
	Done       bool
	Loaded     int
	Total      int
	Stableford bool
}

// Tournaments lists already viewed/stored tournaments sorted by date.
func Tournaments(database *sql.DB) http.HandlerFunc {
	tmpl := template.Must(template.ParseFiles("templates/base.html", "templates/tournaments.html"))
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := db.ListTournaments(database)
		if err != nil {
			http.Error(w, "DB error", 500)
			return
		}
		data := TournamentListData{}
		for _, t := range list {
			data.Tournaments = append(data.Tournaments, TournamentListItem{
				ID:         t.ID,
				CourseName: t.CourseName,
				Date:       formatDateTime(t.Date),
			})
		}
		tmpl.ExecuteTemplate(w, "base", data)
	}
}

// loadTournamentData returns categories and entries, using the DB cache
// so the API is not hit again for already viewed tournaments.
func loadTournamentData(database *sql.DB, client *api.Client, ctx context.Context, tid int64) ([]api.TournamentCategory, []api.TournamentEntry, error) {
	catJSON, entJSON, ok, err := db.GetTournamentCache(database, tid)
	if err != nil {
		return nil, nil, err
	}
	var cats []api.TournamentCategory
	var entries []api.TournamentEntry
	if ok {
		if json.Unmarshal([]byte(catJSON), &cats) == nil &&
			json.Unmarshal([]byte(entJSON), &entries) == nil {
			return sortCategories(cats), entries, nil
		}
	}

	tidStr := strconv.FormatInt(tid, 10)
	cats, err = client.GetTournamentCategories(ctx, tidStr)
	if err != nil {
		return nil, nil, err
	}
	entries, err = client.GetTournamentEntries(ctx, tidStr)
	if err != nil {
		return nil, nil, err
	}
	cb, _ := json.Marshal(cats)
	eb, _ := json.Marshal(entries)
	if err := db.SaveTournamentCache(database, tid, string(cb), string(eb)); err != nil {
		return nil, nil, err
	}
	return sortCategories(cats), entries, nil
}

func sortCategories(cats []api.TournamentCategory) []api.TournamentCategory {
	sort.SliceStable(cats, func(i, j int) bool {
		if cats[i].Main != cats[j].Main {
			return cats[i].Main
		}
		return cats[i].Order < cats[j].Order
	})
	return cats
}

// TournamentResults renders the results page shell with the category
// drop-down; rows are polled from the data endpoint by JS.
func TournamentResults(database *sql.DB, client *api.Client) http.HandlerFunc {
	tmpl := template.Must(template.ParseFiles("templates/base.html", "templates/tournament_results.html"))
	return func(w http.ResponseWriter, r *http.Request) {
		tid, err := strconv.ParseInt(r.PathValue("tid"), 10, 64)
		if err != nil {
			http.Error(w, "Invalid tournament ID", 400)
			return
		}

		cats, _, err := loadTournamentData(database, client, r.Context(), tid)
		if err != nil {
			http.Error(w, "API error: "+err.Error(), 502)
			return
		}
		if len(cats) == 0 {
			http.Error(w, "Tournament has no categories", 404)
			return
		}

		selected, _ := strconv.ParseInt(r.URL.Query().Get("category"), 10, 64)
		found := false
		for _, c := range cats {
			if c.TournamentCategoryID == selected {
				found = true
				break
			}
		}
		if !found {
			selected = cats[0].TournamentCategoryID
		}

		data := TournamentResultsData{
			TournamentID: tid,
			Title:        tournamentTitle(database, tid),
			SelectedID:   selected,
		}
		for _, c := range cats {
			data.Categories = append(data.Categories, CategoryOption{
				ID:       c.TournamentCategoryID,
				Name:     c.Name,
				Selected: c.TournamentCategoryID == selected,
			})
		}
		tmpl.ExecuteTemplate(w, "base", data)
	}
}

func tournamentTitle(database *sql.DB, tid int64) string {
	for _, t := range mustList(database) {
		if t.ID == tid {
			return t.CourseName + " " + formatDateTime(t.Date)
		}
	}
	return strconv.FormatInt(tid, 10)
}

func mustList(database *sql.DB) []db.Tournament {
	list, err := db.ListTournaments(database)
	if err != nil {
		return nil
	}
	return list
}

// --- background result fetching ---

var (
	fetchMu  sync.Mutex
	fetching = map[int64]bool{}
)

// startResultFetch fetches missing golfer results for a tournament in the
// background. Only one job per tournament runs at a time; the API client
// throttles calls so servers are not overloaded.
func startResultFetch(database *sql.DB, client *api.Client, tid int64, golferIDs []int64) {
	fetchMu.Lock()
	if fetching[tid] {
		fetchMu.Unlock()
		return
	}
	fetching[tid] = true
	fetchMu.Unlock()

	go func() {
		defer func() {
			fetchMu.Lock()
			delete(fetching, tid)
			fetchMu.Unlock()
		}()
		tidStr := strconv.FormatInt(tid, 10)
		for _, gid := range golferIDs {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			details, err := client.GetRoundDetail(ctx, tidStr, strconv.FormatInt(gid, 10))
			cancel()
			if err != nil {
				// 404 = golfer has no result yet; other errors are
				// transient and will be retried on the next poll.
				if strings.Contains(err.Error(), "404") {
					db.SaveTournamentResult(database, tid, gid, 0, 0, false)
				}
				continue
			}
			strokes, stableford, has := 0, 0, false
			if len(details) > 0 {
				d := details[0]
				strokes, stableford = d.Strokes, d.StablefordNetto
				has = strokes > 0 || stableford > 0
			}
			db.SaveTournamentResult(database, tid, gid, strokes, stableford, has)
		}
	}()
}

// TournamentResultsRows returns the current result rows as an HTML fragment
// plus a done flag, and kicks off background fetching of missing results.
func TournamentResultsRows(database *sql.DB, client *api.Client) http.HandlerFunc {
	tmpl := template.Must(template.ParseFiles("templates/tournament_results.html"))
	return func(w http.ResponseWriter, r *http.Request) {
		tid, err := strconv.ParseInt(r.PathValue("tid"), 10, 64)
		if err != nil {
			http.Error(w, "Invalid tournament ID", 400)
			return
		}
		catID, _ := strconv.ParseInt(r.URL.Query().Get("category"), 10, 64)

		cats, entries, err := loadTournamentData(database, client, r.Context(), tid)
		if err != nil {
			http.Error(w, "API error: "+err.Error(), 502)
			return
		}

		var category *api.TournamentCategory
		for i := range cats {
			if cats[i].TournamentCategoryID == catID {
				category = &cats[i]
				break
			}
		}
		if category == nil {
			http.Error(w, "Unknown category", 404)
			return
		}

		var members []api.TournamentEntry
		for _, e := range entries {
			for _, c := range e.Categories {
				if c.CategoryID == catID {
					members = append(members, e)
					break
				}
			}
		}

		results, err := db.GetTournamentResults(database, tid)
		if err != nil {
			http.Error(w, "DB error", 500)
			return
		}

		var missing []int64
		for _, m := range members {
			if _, ok := results[m.GolferID]; !ok {
				missing = append(missing, m.GolferID)
			}
		}
		if len(missing) > 0 {
			startResultFetch(database, client, tid, missing)
		}

		data := buildResultRows(*category, members, results)

		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, "result-rows", data); err != nil {
			http.Error(w, "Template error", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"done": data.Done,
			"html": buf.String(),
		})
	}
}

func buildResultRows(category api.TournamentCategory, members []api.TournamentEntry, results map[int64]db.CachedResult) ResultRowsData {
	stablefordSort := category.HcpUse

	type scored struct {
		entry api.TournamentEntry
		res   db.CachedResult
	}
	var ranked []scored
	var rest []api.TournamentEntry // no result yet or still loading
	pending := map[int64]bool{}

	for _, m := range members {
		res, ok := results[m.GolferID]
		if ok && res.HasResult {
			ranked = append(ranked, scored{m, res})
		} else {
			if !ok {
				pending[m.GolferID] = true
			}
			rest = append(rest, m)
		}
	}

	sortKey := func(s scored) int {
		if stablefordSort {
			return -s.res.Stableford // most points first
		}
		return s.res.Strokes // lowest strokes first
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		ki, kj := sortKey(ranked[i]), sortKey(ranked[j])
		if ki != kj {
			return ki < kj
		}
		return ranked[i].entry.GolferName < ranked[j].entry.GolferName
	})
	sort.SliceStable(rest, func(i, j int) bool {
		if rest[i].HcpNumber != rest[j].HcpNumber {
			return rest[i].HcpNumber < rest[j].HcpNumber
		}
		return rest[i].GolferName < rest[j].GolferName
	})

	data := ResultRowsData{
		Total:      len(members),
		Loaded:     len(members) - len(pending),
		Stableford: stablefordSort,
	}
	data.Done = len(pending) == 0

	place := 0
	for i, s := range ranked {
		if i == 0 || sortKey(s) != sortKey(ranked[i-1]) {
			place = i + 1
		}
		data.Rows = append(data.Rows, ResultRow{
			Place:      fmt.Sprintf("%d.", place),
			Name:       s.entry.GolferName,
			Club:       s.entry.ClubShortName,
			HCP:        s.entry.HcpText,
			Strokes:    strconv.Itoa(s.res.Strokes),
			Stableford: strconv.Itoa(s.res.Stableford),
		})
	}
	for _, m := range rest {
		data.Rows = append(data.Rows, ResultRow{
			Place:    "—",
			Name:     m.GolferName,
			Club:     m.ClubShortName,
			HCP:      m.HcpText,
			Pending:  pending[m.GolferID],
			NoResult: !pending[m.GolferID],
		})
	}
	return data
}

// TournamentInvalidate clears the cached categories, entries and results
// for a tournament so they are fetched fresh from the API.
func TournamentInvalidate(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tid, err := strconv.ParseInt(r.PathValue("tid"), 10, 64)
		if err != nil {
			http.Error(w, "Invalid tournament ID", 400)
			return
		}
		if err := db.InvalidateTournament(database, tid); err != nil {
			http.Error(w, "DB error", 500)
			return
		}
		target := fmt.Sprintf("/tournament/%d/results", tid)
		if cat := r.URL.Query().Get("category"); cat != "" {
			target += "?category=" + cat
		}
		http.Redirect(w, r, target, http.StatusSeeOther)
	}
}
