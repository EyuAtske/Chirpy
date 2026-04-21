package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/EyuAtske/Chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db *database.Queries
	platform string
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
	err = server.ListenAndServe()
	fmt.Println(err)
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func handleReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	
	fmt.Fprintf(w, `<html>
					<body>
						<h1>Welcome, Chirpy Admin</h1>
						<p>Chirpy has been visited %d times!</p>
					</body>
					</html>`, cfg.fileserverHits.Load())
}

func (cfg *apiConfig) handleReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Reset is only allowed in dev environment."))
		return
	}
	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
	err := apicfg.db.DeleteUsers(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error while resetting users"))
		return
	}
}

func checkBadWords(body string) string{
	splitBody := strings.Split(body, " ")
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	for _, word := range splitBody {
		wr := strings.ToLower(word)
		for _, badWord := range badWords {
			if wr == badWord {
				body = strings.Replace(body, word, "****", -1)
			}
		}
	}
	return body
}

func handleUsers(w http.ResponseWriter, r *http.Request){
	type email struct{
		Email string `json:"email"`
	}
	type user struct{
		Userid uuid.UUID `json:"id"`
		Created_at time.Time `json:"created_at"`
		Updated_at time.Time `json:"updated_at"`
		Email string `json:"email"`
	}
	var reqEmail email
	decoder := json.NewDecoder(r.Body)
	err:= decoder.Decode(&reqEmail)
	if err != nil{
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error while decoding"))
		return
	}
	users, err := apicfg.db.CreateUser(r.Context(), reqEmail.Email)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error while creating user"))
		return
	}
	respUser := user{
		Userid: users.ID,
		Created_at: users.CreatedAt,
		Updated_at: users.UpdatedAt,
		Email: users.Email,
	}
	data, err := json.Marshal(respUser)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error while encoding response"))
		return
	}
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write(data)
}

func handleChirps(w http.ResponseWriter, r *http.Request){
	type params struct {
		Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	type response struct {
		Id uuid.UUID `json:"id"`
		Created_at time.Time `json:"created_at"`
		Updated_at time.Time `json:"updated_at"`
		Body string `json:"body"`
		UserId uuid.UUID `json:"user_id"`
	}
	type errorResponse struct {
		Error string `json:"error"`
	}
	var p params
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorRep := errorResponse{
			Error: "Something went wrong parsing the request body",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	if len(p.Body) > 140 {
		w.WriteHeader(http.StatusBadRequest)
		errorRep := errorResponse{
			Error: "Chirp is too long",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return 
	}

	cleaned_body:= checkBadWords(p.Body)
	chirp, err := apicfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body: cleaned_body,
		UserID: p.UserID,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorRep := errorResponse{
			Error: "Something went wrong creating the chirp",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	resp := response{
		Id: chirp.ID,
		Created_at: chirp.CreatedAt,
		Updated_at: chirp.UpdatedAt,
		Body: chirp.Body,
		UserId: chirp.UserID,
	}
	respData, err := json.Marshal(resp)
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write(respData)
}