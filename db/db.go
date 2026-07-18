package db

import (
	"database/sql"
	"strings"

	_ "modernc.org/sqlite"
)

func Init(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	return db, db.Ping()
}

func Migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS golfers (
			id   INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS tournaments (
			id          INTEGER PRIMARY KEY,
			course_name TEXT NOT NULL DEFAULT '',
			date        TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS tournament_cache (
			tournament_id   INTEGER PRIMARY KEY,
			categories_json TEXT NOT NULL,
			entries_json    TEXT NOT NULL,
			fetched_at      TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS tournament_results (
			tournament_id INTEGER NOT NULL,
			golfer_id     INTEGER NOT NULL,
			strokes       INTEGER NOT NULL DEFAULT 0,
			stableford    INTEGER NOT NULL DEFAULT 0,
			has_result    INTEGER NOT NULL DEFAULT 0,
			fetched_at    TEXT NOT NULL,
			PRIMARY KEY (tournament_id, golfer_id)
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	// Older databases predate these columns.
	alters := []string{
		`ALTER TABLE tournament_results ADD COLUMN holes_json TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tournaments ADD COLUMN name TEXT NOT NULL DEFAULT ''`,
	}
	for _, s := range alters {
		if _, err := db.Exec(s); err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return err
		}
	}
	return nil
}

type Golfer struct {
	ID   int
	Name string
}

func ListGolfers(db *sql.DB) ([]Golfer, error) {
	rows, err := db.Query(`SELECT id, name FROM golfers ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var golfers []Golfer
	for rows.Next() {
		var g Golfer
		if err := rows.Scan(&g.ID, &g.Name); err != nil {
			return nil, err
		}
		golfers = append(golfers, g)
	}
	return golfers, rows.Err()
}

func GetGolfer(db *sql.DB, id int) (*Golfer, error) {
	var g Golfer
	err := db.QueryRow(`SELECT id, name FROM golfers WHERE id = ?`, id).Scan(&g.ID, &g.Name)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func AddGolfer(db *sql.DB, id int, name string) error {
	_, err := db.Exec(`INSERT INTO golfers (id, name) VALUES (?, ?)`, id, name)
	return err
}

func DeleteGolfer(db *sql.DB, id int) error {
	res, err := db.Exec(`DELETE FROM golfers WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func GolferExists(db *sql.DB, id int) (bool, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM golfers WHERE id = ?`, id).Scan(&count)
	return count > 0, err
}
