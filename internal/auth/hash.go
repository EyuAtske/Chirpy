package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/alexedwards/argon2id"
)

func HashPassword(password string) (string, error){
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func CheckPasswordHash(password, hash string) (bool, error){
	check, err:= argon2id.ComparePasswordAndHash(password, hash)
	return check, err
}

func GetAPIKey(headers http.Header) (string, error){
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing Authorized header")
	}
	splitAuth := strings.Split(authHeader, " ")
	if len(splitAuth) < 2 || splitAuth[0] != "ApiKey" {
		return "", errors.New("malformed authorization header")
	}
	return splitAuth[1], nil
}