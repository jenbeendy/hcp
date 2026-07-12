package db

import (
	"database/sql"

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
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS golfers (
		id   INTEGER PRIMARY KEY,
		name TEXT NOT NULL
	)`)
	return err
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
