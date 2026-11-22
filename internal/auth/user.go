package auth

// User represents an authenticated user.
type User struct {
	Provider string `json:"provider"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`

	// EmailVerified is true when the provider confirms the user owns Email.
	EmailVerified bool `json:"-"`
	// HostedDomain is Google's `hd` claim: the Workspace domain of the account.
	HostedDomain string `json:"-"`
	// Subject is the OIDC `sub` claim, used to match the ID token to userinfo.
	Subject string `json:"-"`
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
