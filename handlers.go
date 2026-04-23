package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/EyuAtske/Chirpy/internal/auth"
	"github.com/EyuAtske/Chirpy/internal/database"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

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

type email struct{
		Password string `json:"password"`
		Email string `json:"email"`
	}

type user struct{
		Userid uuid.UUID `json:"id"`
		Created_at time.Time `json:"created_at"`
		Updated_at time.Time `json:"updated_at"`
		Email string `json:"email"`
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
	var reqEmail email
	decoder := json.NewDecoder(r.Body)
	err:= decoder.Decode(&reqEmail)
	if err != nil{
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error while decoding"))
		return
	}
	hashedPassword, err := auth.HashPassword(reqEmail.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorRep := errorResponse{
			Error: "Error while decoding request",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	users, err := apicfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Email: reqEmail.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorRep := errorResponse{
			Error: "Error while creating user",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
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
		errorRep := errorResponse{
			Error: "Error while encoding response",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
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

func handleGetChirps(w http.ResponseWriter, r *http.Request){
	chirps, err := apicfg.db.GetChirps(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		errorRep := errorResponse{
			Error: "Something went wrong fetching the chirps",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	var resp []response
	for _, chirp := range chirps {
		resp = append(resp, response{
			Id: chirp.ID,
			Created_at: chirp.CreatedAt,
			Updated_at: chirp.UpdatedAt,
			Body: chirp.Body,
			UserId: chirp.UserID,
		})
	}
	data, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		errorRep := errorResponse{
			Error: "Something went wrong encoding the response",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func handleGetSingleChirps(w http.ResponseWriter, r *http.Request){
	chirpIDStr := r.PathValue("chirpID")

    chirpID, _ := uuid.Parse(chirpIDStr)
	chirp, err := apicfg.db.GetChirp(r.Context(), chirpID)
	if err != nil{
		w.WriteHeader(http.StatusNotFound)
		errorRep := errorResponse{
			Error: "Something went wrong when fetching chirp",
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
	respData, _ := json.Marshal(resp)
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(respData)
}

func handleLogIn(w http.ResponseWriter, r *http.Request){
	var reqEmail email
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&reqEmail)
	if err != nil{
		w.WriteHeader(http.StatusInternalServerError)
		errorRep := errorResponse{
			Error: "Error while decoding request",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	usr, err := apicfg.db.GetUserByEmail(r.Context(), reqEmail.Email)
	if err != nil{
		w.WriteHeader(http.StatusUnauthorized)
		errorRep := errorResponse{
			Error: "Incorrect email or password",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	check, _ := auth.CheckPasswordHash(reqEmail.Password, usr.HashedPassword)
	if !check{
		w.WriteHeader(http.StatusUnauthorized)
		errorRep := errorResponse{
			Error: "Incorrect email or password",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	resp := user{
		Userid: usr.ID,
		Created_at: usr.CreatedAt,
		Updated_at: usr.UpdatedAt,
		Email: usr.Email,
	}
	respData, _:= json.Marshal(resp)
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(respData)
}