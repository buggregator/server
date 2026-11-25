package explain

import (
	"regexp"
	"strings"
)

// Words that mark a value as a secret in a key name: password=…,
// "api_key": "…", X-Auth-Token: …, DB_PASSWORD=… Key characters stop at a
// dot, so a file name (VerifyCsrfToken.php:85) is never read as a key.
const secretKey = `[\w-]*(?:password|passwd|pwd|secret|token|api[_-]?key|apikey|access[_-]?key|private[_-]?key|sentry[_-]?key|authorization|cookie|session[_-]?id|credential|dsn|signature)[\w-]*`

const redacted = "[redacted]"

var (
	// Headers whose whole value is a secret (every cookie, any scheme).
	secretHeader = regexp.MustCompile(`(?im)^([ \t]*(?:set-cookie|cookie|authorization|proxy-authorization|x-sentry-auth|` + secretKey + `)[ \t]*:[ \t]*)\S.*$`)
	// Credentials in a URL: a Sentry DSN key, user:password@host.
	urlCreds = regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.-]*://)[^\s/@:"'<>]+(?::[^\s/@"'<>]*)?@`)
	// Authorization schemes with their credentials.
	authScheme = regexp.MustCompile(`(?i)\b(bearer|basic|digest)\s+[A-Za-z0-9._~+/=-]+`)
	// JSON: "secret_key": "value", 123456 or ["value", …].
	jsonSecret = regexp.MustCompile(`(?i)("` + secretKey + `"\s*:\s*)(?:"(?:[^"\\]|\\.)*"|\[[^\]]*\]|-?\d[\d.eE+-]*)`)
	// Text: key=value, key: value (env, query strings, logs).
	kvSecret = regexp.MustCompile(`(?i)\b(` + secretKey + `)(\s*[=:]\s*)[^\s&,;"'<>]+`)
	// Token shapes worth catching wherever they stand.
	tokenShape = regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_-]{16,}|gh[pousr]_[A-Za-z0-9]{20,}|xox[abposr]-[A-Za-z0-9-]{10,}|AKIA[0-9A-Z]{16}|h2a_[A-Za-z0-9_-]{16,}|eyJ[\w-]{8,}\.[\w-]{8,}\.[\w-]{8,})`)
)

// scrub masks obvious secrets in a text that leaves this machine. It errs on
// the side of masking, but keeps the error itself: an exception's name with
// a secret word (TokenMismatchException: …) is not a key.
func scrub(s string) string {
	s = secretHeader.ReplaceAllStringFunc(s, func(m string) string {
		sub := secretHeader.FindStringSubmatch(m)
		if isErrorName(strings.TrimRight(strings.TrimSpace(sub[1]), ": \t")) {
			return m
		}
		return sub[1] + redacted
	})
	s = urlCreds.ReplaceAllString(s, "${1}"+redacted+"@")
	s = authScheme.ReplaceAllString(s, "${1} "+redacted)
	s = jsonSecret.ReplaceAllString(s, `${1}"`+redacted+`"`)
	s = kvSecret.ReplaceAllStringFunc(s, func(m string) string {
		sub := kvSecret.FindStringSubmatch(m)
		if strings.Contains(sub[2], ":") && isErrorName(sub[1]) {
			return m
		}
		return sub[1] + sub[2] + redacted
	})
	return tokenShape.ReplaceAllString(s, redacted)
}

// isErrorName: TokenMismatchException, InvalidCredentialsError — a class
// name before its message, not a key before its value.
func isErrorName(key string) bool {
	k := strings.ToLower(key)
	return strings.HasSuffix(k, "exception") || strings.HasSuffix(k, "error")
}
