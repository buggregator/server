package explain

import "regexp"

// Words that mark a value as a secret in a key name: password=…,
// "api_key": "…", X-Auth-Token: …, DB_PASSWORD=…
const secretKey = `[\w.-]*(?:password|passwd|pwd|secret|token|api[_-]?key|apikey|access[_-]?key|private[_-]?key|sentry[_-]?key|authorization|cookie|session[_-]?id|credential|dsn|signature)[\w.-]*`

const redacted = "[redacted]"

var scrubRules = []struct {
	re   *regexp.Regexp
	with string
}{
	// Credentials in a URL: a Sentry DSN key, user:password@host.
	{regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.-]*://)[^\s/@:"'<>]+(?::[^\s/@"'<>]*)?@`), "${1}" + redacted + "@"},
	// Authorization schemes with their credentials.
	{regexp.MustCompile(`(?i)\b(bearer|basic|digest|token)\s+[A-Za-z0-9._~+/=-]{8,}`), "${1} " + redacted},
	// JSON: "secret_key": "value" or "cookie": ["value", …].
	{regexp.MustCompile(`(?i)("` + secretKey + `"\s*:\s*)(?:"(?:[^"\\]|\\.)*"|\[[^\]]*\])`), `${1}"` + redacted + `"`},
	// Text: key=value, key: value (headers, env, query strings, logs).
	{regexp.MustCompile(`(?i)\b(` + secretKey + `)(\s*[=:]\s*)[^\s&,;"'<>]+`), "${1}${2}" + redacted},
	// Token shapes worth catching wherever they stand.
	{regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_-]{16,}|gh[pousr]_[A-Za-z0-9]{20,}|xox[abposr]-[A-Za-z0-9-]{10,}|AKIA[0-9A-Z]{16}|h2a_[A-Za-z0-9_-]{16,}|eyJ[\w-]{8,}\.[\w-]{8,}\.[\w-]{8,})`), redacted},
}

// scrub masks obvious secrets in a text that leaves this machine. It errs on
// the side of masking: an over-masked value costs the agent a detail, a
// missed one leaks a key.
func scrub(s string) string {
	for _, r := range scrubRules {
		s = r.re.ReplaceAllString(s, r.with)
	}
	return s
}
