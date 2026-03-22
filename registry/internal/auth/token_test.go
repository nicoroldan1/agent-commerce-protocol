package auth

import "testing"

func TestGenerateToken(t *testing.T) {
	token := GenerateToken()
	if len(token) < 10 {
		t.Fatal("token too short")
	}
	if token[:4] != "rgt_" {
		t.Fatalf("expected rgt_ prefix, got %s", token[:4])
	}
}

func TestHashAndValidate(t *testing.T) {
	token := GenerateToken()
	hash := HashToken(token)

	if !ValidateToken(token, hash) {
		t.Fatal("expected valid token")
	}
	if ValidateToken("wrong_token", hash) {
		t.Fatal("expected invalid token")
	}
}
