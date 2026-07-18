package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/hcp/golf/api"
	"github.com/hcp/golf/db"
	"github.com/hcp/golf/handlers"
)

func main() {
	godotenv.Load()

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./golf.db"
	}

	database, err := db.Init(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		log.Fatal(err)
	}

	client := api.NewClient(os.Getenv("CGF_TOKEN"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("GET /", handlers.Players(database))
	mux.HandleFunc("GET /golfer/{id}", handlers.Golfer(database, client))
	mux.HandleFunc("GET /golfer/{id}/registrations", handlers.Registrations(database, client))
	mux.HandleFunc("GET /golfer/{id}/tournament/{tid}/detail", handlers.RoundDetail(database, client))
	mux.HandleFunc("GET /tournament", handlers.Tournaments(database))
	mux.HandleFunc("GET /tournament/{tid}/results", handlers.TournamentResults(database, client))
	mux.HandleFunc("GET /tournament/{tid}/results/data", handlers.TournamentResultsRows(database, client))
	mux.HandleFunc("POST /tournament/{tid}/invalidate", handlers.TournamentInvalidate(database))
	mux.HandleFunc("POST /tournament/{tid}/delete", handlers.TournamentDelete(database))
	mux.HandleFunc("GET /adminpage", handlers.AdminGet(database))
	mux.HandleFunc("POST /adminpage", handlers.AdminPost(database, client))
	mux.HandleFunc("POST /adminpage/delete", handlers.AdminDelete(database))

	log.Printf("Listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
