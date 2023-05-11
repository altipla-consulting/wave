package env

import (
	"github.com/altipla-consulting/env"
)

func SentryAuthToken() string {
	return env.MustRead("SENTRY_AUTH_TOKEN")
}

func GoogleProject() string {
	return env.MustRead("GOOGLE_PROJECT")
}
