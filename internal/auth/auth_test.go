package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWT(t *testing.T) {
	secret := "mysecret"
	userID := uuid.New()
	expiresIn := time.Hour

	// Test token creation
	token, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("Failed to create JWT: %v", err)
	}

	// Test token validation
	returnedUserID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("Failed to validate JWT: %v", err)
	}
	if returnedUserID != userID {
		t.Fatalf("Expected user ID %v, got %v", userID, returnedUserID)
	}

	// Test expired token
	expiredToken, err := MakeJWT(userID, secret, -time.Hour)
	if err != nil {
		t.Fatalf("Failed to create expired JWT: %v", err)
	}
	_, err = ValidateJWT(expiredToken, secret)
	if err == nil {
		t.Fatal("Expected error for expired token, got none")
	}

	// Test invalid token
	_, err = ValidateJWT("invalidtoken", secret)
	if err == nil {
		t.Fatal("Expected error for invalid token, got none")
	}
}

func TestGetBearerToken(t *testing.T) {
	headers := make(map[string][]string)
	headers["Authorization"] = []string{"Bearer mytoken"}
	token, err:= GetBearerToken(headers)
	if err != nil {
		t.Fatalf("Failed to get bearer token: %v", err)
	}
	if token != "mytoken" {
		t.Fatalf("Expected token 'mytoken', got '%s'", token)
	}

	// Test missing header
	headers = make(map[string][]string)
	_, err = GetBearerToken(headers)
	if err == nil {
		t.Fatal("Expected error for missing Authorization header, got none")
	}
}