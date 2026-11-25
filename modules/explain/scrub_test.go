package explain

import (
	"strings"
	"testing"
)

func TestScrub(t *testing.T) {
	tests := []struct {
		name, in string
		gone     []string // must not survive
		kept     []string // must survive
	}{
		{"sentry dsn", `dsn https://abc123def456@o1.ingest.sentry.io/42`, []string{"abc123def456"}, []string{"o1.ingest.sentry.io/42"}},
		{"url user and password", `mysql://root:hunter2@db:3306/app`, []string{"hunter2", "root:"}, []string{"db:3306/app"}},
		{"authorization header line", `Authorization: Bearer eyJhbGciOi.payload123.sig45678`, []string{"eyJhbGciOi", "payload123"}, []string{"Authorization"}},
		{"basic auth", `authorization: Basic dXNlcjpwYXNzd29yZA==`, []string{"dXNlcjpwYXNzd29yZA"}, nil},
		{"json header list", `{"authorization":["Bearer abcdefgh12345678"],"accept":["*/*"]}`, []string{"abcdefgh12345678"}, []string{`"accept":["*/*"]`}},
		{"json password", `{"user":"bob","password":"s3cr3t!"}`, []string{"s3cr3t!"}, []string{`"user":"bob"`}},
		{"json api key", `{"api_key": "k-123456"}`, []string{"k-123456"}, nil},
		{"query string", `GET /login?user=bob&password=hunter2&next=/`, []string{"hunter2"}, []string{"user=bob", "next=/"}},
		{"env", `DB_PASSWORD=hunter2 APP_ENV=local AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI`, []string{"hunter2", "wJalrXUtnFEMI"}, []string{"APP_ENV=local"}},
		{"x-sentry-auth", `X-Sentry-Auth: Sentry sentry_key=abc123, sentry_version=7`, []string{"abc123"}, []string{"X-Sentry-Auth: [redacted]"}},
		{"cookie", `Cookie: laravel_session=eyJpdiI6IkFi`, []string{"laravel_session=eyJpdiI6IkFi"}, nil},
		{"token shapes", `keys sk-proj-ABCDEFGHIJKLMNOPQRST ghp_ABCDEFGHIJKLMNOPQRSTUV AKIAABCDEFGHIJKLMNOP h2a_TESTtokenTESTtoken`, []string{"sk-proj-ABC", "ghp_ABC", "AKIAABCD", "h2a_TEST"}, nil},
		{"every cookie", `Cookie: XSRF-TOKEN=eyJabc; laravel_session=eyJpdiI6SESSION; PHPSESSID=abcdef0123456789`, []string{"eyJabc", "eyJpdiI6SESSION", "abcdef0123456789"}, []string{"Cookie: [redacted]"}},
		{"cookie header indented", "Headers:\n  Cookie: a=1; PHPSESSID=abcdef0123456789\n  Accept: */*", []string{"PHPSESSID", "abcdef0123456789"}, []string{"Accept: */*"}},
		{"set-cookie", `Set-Cookie: laravel_session=abc123; path=/; httponly`, []string{"abc123", "laravel_session"}, nil},
		{"short basic", `Authorization: Basic abc`, []string{"abc"}, []string{"Authorization: [redacted]"}},
		{"proxy authorization", `Proxy-Authorization: Negotiate YIIabc`, []string{"YIIabc"}, nil},
		{"json numeric secret", `{"password": 123456, "id": 7}`, []string{"123456"}, []string{`"id": 7`}},
		{"frame in a token file", `  at app/Http/Middleware/VerifyCsrfToken.php:85 in handle`, nil, []string{"VerifyCsrfToken.php:85 in handle"}},
		{"frames in secret-word files", `at PasswordBroker.php:42 at EncryptCookies.php:7 at ValidateSignature.php:19 at CredentialStore.php:3`, nil, []string{"PasswordBroker.php:42", "EncryptCookies.php:7", "ValidateSignature.php:19", "CredentialStore.php:3"}},
		{"csrf mismatch", `Illuminate\Session\TokenMismatchException: CSRF token mismatch.`, nil, []string{`TokenMismatchException: CSRF token mismatch.`}},
		{"aws invalid token", `Aws\Exception\InvalidCredentialsException: The security token is invalid`, nil, []string{"InvalidCredentialsException: The security token is invalid"}},
		{"error class line", `Error: TokenExpiredError: jwt expired`, nil, []string{"TokenExpiredError: jwt expired"}},
		{"dotted config key", `spring.datasource.password=hunter2`, []string{"hunter2"}, nil},
		{"plain error kept", `Order 15 not found in OrderService::load at app/Order.php:42`, nil, []string{"Order 15 not found in OrderService::load at app/Order.php:42"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := scrub(tt.in)
			for _, g := range tt.gone {
				if strings.Contains(out, g) {
					t.Errorf("%q survived: %s", g, out)
				}
			}
			for _, k := range tt.kept {
				if !strings.Contains(out, k) {
					t.Errorf("%q lost: %s", k, out)
				}
			}
		})
	}
}
