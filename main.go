package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/EyuAtske/Chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db *database.Queries
	platform string
	secret string
}

var apicfg apiConfig

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL must be set")
	}

	dbConn, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error opening database: %s", err)
	}
	dbQueries := database.New(dbConn)
	apicfg.fileserverHits = atomic.Int32{}
	apicfg.db = dbQueries
	apicfg.platform = os.Getenv("PLATFORM")
	apicfg.secret = os.Getenv("SECRET_KEY")
	servermux := http.NewServeMux()
	servermux.Handle("/app/", http.StripPrefix("/app", apicfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	server := &http.Server{
		Handler: servermux,
		Addr: ":8080",
	}
	servermux.HandleFunc("GET /api/healthz", handleReadiness)
	servermux.HandleFunc("GET /admin/metrics", apicfg.handleMetrics)
	servermux.HandleFunc("POST /admin/reset", apicfg.handleReset)
	servermux.HandleFunc("POST /api/users", handleUsers)
	servermux.HandleFunc("POST /api/chirps", handleChirps)
	servermux.HandleFunc("GET /api/chirps", handleGetChirps)
	servermux.HandleFunc("GET /api/chirps/{chirpID}", handleGetSingleChirps)
	servermux.HandleFunc("POST /api/login", handleLogIn)
	servermux.HandleFunc("POST /api/refresh", handleRefresh)
	servermux.HandleFunc("POST /api/revoke", handleRevoke)
	servermux.HandleFunc("PUT /api/users", handleUpdates)
	servermux.HandleFunc("DELETE /api/chirps/{chirpID}", handleDelete)
	err = server.ListenAndServe()
	fmt.Println(err)
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

