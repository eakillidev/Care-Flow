package auth

import "testing"

func TestPasswordHashAndVerification(t *testing.T) {
	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == "correct-password" {
		t.Fatal("password hash must not equal plaintext")
	}
	if !VerifyPassword(hash, "correct-password") {
		t.Fatal("expected correct password to verify")
	}
	if VerifyPassword(hash, "incorrect-password") {
		t.Fatal("expected incorrect password to fail verification")
	}
}
