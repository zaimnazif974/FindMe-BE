package middlewares

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIssueAndValidateToken(t *testing.T) {
	userID := uuid.New()
	token, err := IssueToken(userID, "this-is-a-long-test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ValidateToken(token, "this-is-a-long-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if got != userID {
		t.Fatalf("ValidateToken() = %s, want %s", got, userID)
	}
}

func TestValidateTokenRejectsWrongSecret(t *testing.T) {
	token, err := IssueToken(uuid.New(), "this-is-a-long-test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateToken(token, "different-long-test-secret"); err == nil {
		t.Fatal("ValidateToken() accepted a token signed with another secret")
	}
}
