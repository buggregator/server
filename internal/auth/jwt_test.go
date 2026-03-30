package auth

import (
	"testing"
)

func TestTokenStore_CreateAndValidate(t *testing.T) {
	store := NewTokenStore("test-secret-key-32-chars-long!!!")

	user := User{
		Provider: "auth0",
		Username: "john",
		Email:    "john@example.com",
		Avatar:   "https://example.com/avatar.png",
	}

	token, err := store.Create(user)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if token == "" {
		t.Fatal("Create() returned empty token")
	}

	got, err := store.Validate(token)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if got.Provider != user.Provider {
		t.Errorf("Provider = %q, want %q", got.Provider, user.Provider)
	}
	if got.Username != user.Username {
		t.Errorf("Username = %q, want %q", got.Username, user.Username)
	}
	if got.Email != user.Email {
		t.Errorf("Email = %q, want %q", got.Email, user.Email)
	}
	if got.Avatar != user.Avatar {
		t.Errorf("Avatar = %q, want %q", got.Avatar, user.Avatar)
	}
}

func TestTokenStore_ValidateInvalid(t *testing.T) {
	store := NewTokenStore("test-secret")

	_, err := store.Validate("invalid-token")
	if err == nil {
		t.Fatal("Validate() expected error for invalid token")
	}
}

func TestTokenStore_ValidateWrongSecret(t *testing.T) {
	store1 := NewTokenStore("secret-one")
	store2 := NewTokenStore("secret-two")

	user := User{Provider: "test", Username: "john"}
	token, _ := store1.Create(user)

	_, err := store2.Validate(token)
	if err == nil {
		t.Fatal("Validate() expected error for wrong secret")
	}
}
