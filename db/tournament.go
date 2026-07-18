package db

import (
	"database/sql"
	"encoding/json"
	"time"
)

type Tournament struct {
	ID         int64
	CourseName string
	Date       string
}

func UpsertTournament(db *sql.DB, id int64, courseName, date string) error {
	_, err := db.Exec(`INSERT INTO tournaments (id, course_name, date) VALUES (?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET course_name = excluded.course_name, date = excluded.date`,
		id, courseName, date)
	return err
}

func ListTournaments(db *sql.DB) ([]Tournament, error) {
	rows, err := db.Query(`SELECT id, course_name, date FROM tournaments ORDER BY date DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tournament
	for rows.Next() {
		var t Tournament
		if err := rows.Scan(&t.ID, &t.CourseName, &t.Date); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func GetTournamentCache(db *sql.DB, tournamentID int64) (categoriesJSON, entriesJSON string, ok bool, err error) {
	err = db.QueryRow(`SELECT categories_json, entries_json FROM tournament_cache WHERE tournament_id = ?`,
		tournamentID).Scan(&categoriesJSON, &entriesJSON)
	if err == sql.ErrNoRows {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	return categoriesJSON, entriesJSON, true, nil
}

func SaveTournamentCache(db *sql.DB, tournamentID int64, categoriesJSON, entriesJSON string) error {
	_, err := db.Exec(`INSERT INTO tournament_cache (tournament_id, categories_json, entries_json, fetched_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(tournament_id) DO UPDATE SET
			categories_json = excluded.categories_json,
			entries_json = excluded.entries_json,
			fetched_at = excluded.fetched_at`,
		tournamentID, categoriesJSON, entriesJSON, time.Now().Format(time.RFC3339))
	return err
}

// HoleScore is one hole's result, stored in play order for tie-breaking.
type HoleScore struct {
	Strokes    int `json:"s"`
	Stableford int `json:"p"`
}

type CachedResult struct {
	GolferID   int64
	Strokes    int
	Stableford int
	HasResult  bool
	Holes      []HoleScore
}

func GetTournamentResults(db *sql.DB, tournamentID int64) (map[int64]CachedResult, error) {
	rows, err := db.Query(`SELECT golfer_id, strokes, stableford, has_result, holes_json
		FROM tournament_results WHERE tournament_id = ?`, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int64]CachedResult)
	for rows.Next() {
		var r CachedResult
		var holesJSON string
		if err := rows.Scan(&r.GolferID, &r.Strokes, &r.Stableford, &r.HasResult, &holesJSON); err != nil {
			return nil, err
		}
		if holesJSON != "" {
			if err := json.Unmarshal([]byte(holesJSON), &r.Holes); err != nil {
				return nil, err
			}
		}
		out[r.GolferID] = r
	}
	return out, rows.Err()
}

func SaveTournamentResult(db *sql.DB, tournamentID, golferID int64, strokes, stableford int, hasResult bool, holes []HoleScore) error {
	holesJSON := ""
	if len(holes) > 0 {
		b, err := json.Marshal(holes)
		if err != nil {
			return err
		}
		holesJSON = string(b)
	}
	_, err := db.Exec(`INSERT INTO tournament_results (tournament_id, golfer_id, strokes, stableford, has_result, holes_json, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(tournament_id, golfer_id) DO UPDATE SET
			strokes = excluded.strokes,
			stableford = excluded.stableford,
			has_result = excluded.has_result,
			holes_json = excluded.holes_json,
			fetched_at = excluded.fetched_at`,
		tournamentID, golferID, strokes, stableford, hasResult, holesJSON, time.Now().Format(time.RFC3339))
	return err
}

func InvalidateTournament(db *sql.DB, tournamentID int64) error {
	if _, err := db.Exec(`DELETE FROM tournament_cache WHERE tournament_id = ?`, tournamentID); err != nil {
		return err
	}
	_, err := db.Exec(`DELETE FROM tournament_results WHERE tournament_id = ?`, tournamentID)
	return err
}
