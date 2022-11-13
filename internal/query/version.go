package query

import (
	"os"
	"path"
	"time"

	"github.com/altipla-consulting/wave/internal/gerrit"
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

	// Custom override.
	if ref := os.Getenv("WAVE_VERSION"); ref != "" {
		version = ref
	}

	return version
}

func VersionHostname(override string) string {
	if override != "" {
		return override
	}
	if os.Getenv("BUILD_CAUSE") == "SCMTRIGGER" {
		return gerrit.Descriptor()
	}
	return ""
}

func IsRelease() bool {
	return os.Getenv("GITHUB_REF_TYPE") == "tag" || !gerrit.IsPreview()
}
