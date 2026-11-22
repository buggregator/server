package auth

import "strings"

// AccessPolicy decides who may sign in after the provider's login.
// Empty lists let in everyone the provider lets in.
type AccessPolicy struct {
	emails  map[string]bool
	domains map[string]bool
}

// NewAccessPolicy builds a policy from allowed emails and domains
// (case-insensitive; a leading "@" on a domain is ignored).
func NewAccessPolicy(emails, domains []string) AccessPolicy {
	p := AccessPolicy{emails: map[string]bool{}, domains: map[string]bool{}}
	for _, e := range emails {
		if e = strings.ToLower(strings.TrimSpace(e)); e != "" {
			p.emails[e] = true
		}
	}
	for _, d := range domains {
		if d = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(d), "@")); d != "" {
			p.domains[d] = true
		}
	}
	return p
}

// Restricted reports whether any allow-list is set.
func (p AccessPolicy) Restricted() bool {
	return len(p.emails) > 0 || len(p.domains) > 0
}

// Allows reports whether the user may sign in. With a list set, the email
// must be verified by the provider. A domain matches Google's `hd` claim
// (the Workspace the account belongs to) for provider google, the domain of
// the verified email for the others.
func (p AccessPolicy) Allows(u User) bool {
	if !p.Restricted() {
		return true
	}
	email := strings.ToLower(strings.TrimSpace(u.Email))
	if email == "" || !u.EmailVerified {
		return false
	}
	if p.emails[email] {
		return true
	}
	if len(p.domains) == 0 {
		return false
	}
	domain := ""
	if u.Provider == "google" {
		domain = strings.ToLower(u.HostedDomain)
	} else if at := strings.LastIndex(email, "@"); at >= 0 {
		domain = email[at+1:]
	}
	return domain != "" && p.domains[domain]
}
