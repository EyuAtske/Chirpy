package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"sort"

	"github.com/EyuAtske/Chirpy/internal/auth"
	"github.com/EyuAtske/Chirpy/internal/database"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type response struct {
	Id         uuid.UUID `json:"id"`
	Created_at time.Time `json:"created_at"`
	Updated_at time.Time `json:"updated_at"`
	Body       string    `json:"body"`
	UserId     uuid.UUID `json:"user_id"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type email struct {
	Password string `json:"password"`
	Email    string `json:"email"`
}

type user struct {
	Userid     uuid.UUID `json:"id"`
	Created_at time.Time `json:"created_at"`
	Updated_at time.Time `json:"updated_at"`
	Email      string    `json:"email"`
	Is_chirpy_red bool   `json:"is_chirpy_red"`
}

type loginResponse struct {
	Userid     uuid.UUID `json:"id"`
	Created_at time.Time `json:"created_at"`
	Updated_at time.Time `json:"updated_at"`
	Email      string    `json:"email"`
	Is_chirpy_red bool   `json:"is_chirpy_red"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
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

func checkBadWords(body string) string {
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

func handleUsers(w http.ResponseWriter, r *http.Request) {
	var reqEmail email
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&reqEmail)
	if err != nil {
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
		Email:          reqEmail.Email,
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
		Userid:     users.ID,
		Created_at: users.CreatedAt,
		Updated_at: users.UpdatedAt,
		Email:      users.Email,
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

func handleChirps(w http.ResponseWriter, r *http.Request) {
	type params struct {
		Body string `json:"body"`
	}
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		errorRep := errorResponse{
			Error: "Missing or invalid Authorization header",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}

	id, err := auth.ValidateJWT(bearerToken, apicfg.secret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		errorRep := errorResponse{
			Error: fmt.Sprintf("Invalid token %v", err),
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}

	var p params
	err = json.NewDecoder(r.Body).Decode(&p)
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

	cleaned_body := checkBadWords(p.Body)
	chirp, err := apicfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   cleaned_body,
		UserID: id,
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
		Id:         chirp.ID,
		Created_at: chirp.CreatedAt,
		Updated_at: chirp.UpdatedAt,
		Body:       chirp.Body,
		UserId:     chirp.UserID,
	}
	respData, err := json.Marshal(resp)
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write(respData)
}

func handleGetChirps(w http.ResponseWriter, r *http.Request) {
	s := r.URL.Query().Get("author_id")
	so := r.URL.Query().Get(("sort"))
	var data []byte
	var resp []response
	if s == ""{
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
		for _, chirp := range chirps {
			resp = append(resp, response{
				Id:         chirp.ID,
				Created_at: chirp.CreatedAt,
				Updated_at: chirp.UpdatedAt,
				Body:       chirp.Body,
				UserId:     chirp.UserID,
			})
		}
		if so == "desc"{
			sort.Slice(resp, func(i, j int) bool {return resp[i].Created_at.After(resp[j].Created_at) })
		}
		data, err = json.Marshal(resp)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			errorRep := errorResponse{
				Error: "Something went wrong encoding the response",
			}
			data, _ := json.Marshal(errorRep)
			w.Write(data)
			return
		}
	}else{
		id, err := uuid.Parse(s)
		if err != nil{
			w.WriteHeader(http.StatusNotFound)
				errorRep := errorResponse{
					Error: "Something went wrong parsing aurthor id",
				}
				data, _ := json.Marshal(errorRep)
				w.Write(data)
				return
		}
		chirps, err := apicfg.db.GetChirpsByUserId(r.Context(), id)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			errorRep := errorResponse{
				Error: fmt.Sprintf("Something went wrong fetching the chirps %v", err),
			}
			data, _ := json.Marshal(errorRep)
			w.Write(data)
			return
		}
		for _, chirp := range chirps {
			resp = append(resp, response{
				Id:         chirp.ID,
				Created_at: chirp.CreatedAt,
				Updated_at: chirp.UpdatedAt,
				Body:       chirp.Body,
				UserId:     chirp.UserID,
			})
		}
		if so == "desc"{
			sort.Slice(resp, func(i, j int) bool {return resp[i].Created_at.After(resp[j].Created_at) })
		}
		data, err = json.Marshal(resp)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			errorRep := errorResponse{
				Error: "Something went wrong encoding the response",
			}
			data, _ := json.Marshal(errorRep)
			w.Write(data)
			return
		}
	}
	
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func handleGetSingleChirps(w http.ResponseWriter, r *http.Request) {
	chirpIDStr := r.PathValue("chirpID")

	chirpID, _ := uuid.Parse(chirpIDStr)
	chirp, err := apicfg.db.GetChirp(r.Context(), chirpID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		errorRep := errorResponse{
			Error: "Something went wrong when fetching chirp",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	resp := response{
		Id:         chirp.ID,
		Created_at: chirp.CreatedAt,
		Updated_at: chirp.UpdatedAt,
		Body:       chirp.Body,
		UserId:     chirp.UserID,
	}
	respData, _ := json.Marshal(resp)
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(respData)
}

func handleLogIn(w http.ResponseWriter, r *http.Request) {
	var reqEmail email
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&reqEmail)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorRep := errorResponse{
			Error: "Error while decoding request",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	expires_in_seconds := 3600
	usr, err := apicfg.db.GetUserByEmail(r.Context(), reqEmail.Email)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		errorRep := errorResponse{
			Error: "Incorrect email or password",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	token, err := auth.MakeJWT(usr.ID, apicfg.secret, time.Duration(expires_in_seconds)*time.Second)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorRep := errorResponse{
			Error: "Error while creating token",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}

	refToken := auth.MakeRefreshToken()
	_, err = apicfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:  refToken,
		UserID: usr.ID,
	})

	check, _ := auth.CheckPasswordHash(reqEmail.Password, usr.HashedPassword)
	if !check {
		w.WriteHeader(http.StatusUnauthorized)
		errorRep := errorResponse{
			Error: "Incorrect email or password",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	resp := loginResponse{
		Token: token,
		Userid:     usr.ID,
		Created_at: usr.CreatedAt,
		Updated_at: usr.UpdatedAt,
		Email:      usr.Email,
		Is_chirpy_red: usr.IsChirpyRed,
		RefreshToken: refToken,
	}
	respData, _ := json.Marshal(resp)
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(respData)
}

func handleRefresh(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		errorRep := errorResponse{
			Error: "Missing or invalid Authorization header",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	refToken, err := apicfg.db.GetRefreshToken(r.Context(), bearerToken)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		errorRep := errorResponse{
			Error: "Invalid refresh token",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	if refToken.ExpiresAt.Before(time.Now()) || refToken.RevokedAt.Valid {
		w.WriteHeader(http.StatusUnauthorized)
		errorRep := errorResponse{
			Error: "Refresh token has expired or been revoked",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	newToken, err := auth.MakeJWT(refToken.UserID, apicfg.secret, time.Hour)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorRep := errorResponse{
			Error: "Error while creating token",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	resp := struct {
		Token string `json:"token"`
	}{
		Token: newToken,
	}
	respData, _ := json.Marshal(resp)
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(respData)
}

func handleRevoke(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		errorRep := errorResponse{
			Error: "Missing or invalid Authorization header",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	err = apicfg.db.RevokeRefreshToken(r.Context(), bearerToken)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorRep := errorResponse{
			Error: "Error while revoking refresh token",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNoContent)
}

func handleUpdates(w http.ResponseWriter, r *http.Request) {
	type updateParams struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		errorRep := errorResponse{
			Error: "Missing or invalid Authorization header",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	id, err := auth.ValidateJWT(bearerToken, apicfg.secret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		errorRep := errorResponse{
			Error: fmt.Sprintf("Invalid token %v", err),
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	var params updateParams
	err = json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorRep := errorResponse{
			Error: "Something went wrong parsing the request body",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorRep := errorResponse{
			Error: "Error while hashing password",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	usr, err := apicfg.db.UpdateUserPasswordAndEmail(r.Context(), database.UpdateUserPasswordAndEmailParams{
		HashedPassword: hashedPassword,
		Email:          params.Email,
		ID:             id,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorRep := errorResponse{
			Error: "Error while updating user",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	resp := user{
		Userid:     usr.ID,
		Created_at: usr.CreatedAt,
		Updated_at: usr.UpdatedAt,
		Email:      usr.Email,
		Is_chirpy_red: usr.IsChirpyRed,
	}
	data, err := json.Marshal(resp)
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
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		errorRep := errorResponse{
			Error: fmt.Sprintf("Missing or invalid Authorization header, %v", err),
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	id, err := auth.ValidateJWT(bearerToken, apicfg.secret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		errorRep := errorResponse{
			Error: fmt.Sprintf("Invalid token %v", err),
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	chirp, err := apicfg.db.GetChirp(r.Context(), uuid.MustParse(r.PathValue("chirpID")))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		errorRep := errorResponse{
			Error: "Chirp not found",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	if chirp.UserID != id {
		w.WriteHeader(http.StatusForbidden)
		errorRep := errorResponse{
			Error: "You do not have permission to delete this chirp",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	chirpIdstr := r.PathValue("chirpID")
	chirpID, _ := uuid.Parse(chirpIdstr)
	err = apicfg.db.DeleteChirp(r.Context(), chirpID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		errorRep := errorResponse{
			Error: "Error while deleting chirp",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNoContent)
}

func handlePolka(w http.ResponseWriter, r *http.Request){
	key, err := auth.GetAPIKey(r.Header)
	if err != nil{
		w.WriteHeader(http.StatusUnauthorized)
		errorRep := errorResponse{
			Error: fmt.Sprintf("Missing or invalid Authorization header, %v", err),
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	if key != apicfg.polka{
		w.WriteHeader(http.StatusUnauthorized)
		errorRep := errorResponse{
			Error: "UnAuthorized key",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	type params struct{
		Event string `json:"event"`
		Data struct{
			UserId string `json:"user_id"`
		} `json:"data"`
	}
	var req params
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&req)
	if err != nil{
		w.WriteHeader(http.StatusBadRequest)
		errorRep := errorResponse{
			Error: "Something went wrong parsing the request body",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	if req.Event != "user.upgraded"{
		w.WriteHeader(http.StatusNoContent)
		errorRep := errorResponse{
			Error: "Event not important",
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	userId, _ := uuid.Parse(req.Data.UserId)
	_, err = apicfg.db.SetUserChirpyRed(r.Context(), userId)
	if err != nil{
		w.WriteHeader(http.StatusNotFound)
		errorRep := errorResponse{
			Error: fmt.Sprintf("Error setting chirpy red %v", err),
		}
		data, _ := json.Marshal(errorRep)
		w.Write(data)
		return
	}
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNoContent)
}