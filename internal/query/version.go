package query

import (
	"os"
	"path"
	"time"
)

func Version() string {
	// Default tag for previews and PRs.
	version := time.Now().Format("20060102") + "." + os.Getenv("BUILD_NUMBER")
	if os.Getenv("BUILD_CAUSE") == "SCMTRIGGER" {
		version += ".preview"
	}

	// GitHub releases.
	if ref := os.Getenv("GITHUB_REF"); ref != "" {
		version = path.Base(ref)
	}

	// Gerrit tags.
	if ref := os.Getenv("GERRIT_REFNAME"); ref != "" {
		version = path.Base(ref)
	}

	return version
}
