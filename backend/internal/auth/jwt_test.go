package auth

import (
	"testing"
	"time"
)

func TestGenerate_NonEmpty(t *testing.T) {
	jm := NewJWTManager("test-secret", 72*time.Hour)
	token, err := jm.Generate(1, "a@b.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestValidate_Success(t *testing.T) {
	jm := NewJWTManager("test-secret", 72*time.Hour)
	token, err := jm.Generate(42, "user@example.com")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	claims, err := jm.Validate(token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.UserID != 42 {
		t.Errorf("UserID = %d, want 42", claims.UserID)
	}
	if claims.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", claims.Email, "user@example.com")
	}
	if claims.Issuer != "rakutao" {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, "rakutao")
	}
}

func TestValidate_Expired(t *testing.T) {
	jm := NewJWTManager("test-secret", -1*time.Hour)
	token, err := jm.Generate(1, "a@b.com")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	_, err = jm.Validate(token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestValidate_WrongSecret(t *testing.T) {
	jm1 := NewJWTManager("secret-one", 72*time.Hour)
	token, err := jm1.Generate(1, "a@b.com")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	jm2 := NewJWTManager("secret-two", 72*time.Hour)
	_, err = jm2.Validate(token)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestValidate_InvalidString(t *testing.T) {
	jm := NewJWTManager("test-secret", 72*time.Hour)
	_, err := jm.Validate("not-a-jwt")
	if err == nil {
		t.Fatal("expected error for invalid token string")
	}
}
