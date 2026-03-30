package auth

// User represents an authenticated user.
type User struct {
	Provider string `json:"provider"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
}

// GuestUser returns the default guest user when auth is disabled.
func GuestUser() User {
	return User{
		Provider: "guest",
		Username: "Guest",
		Email:    "",
		Avatar:   "",
	}
}
