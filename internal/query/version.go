package query

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/altipla-consulting/wave/internal/gerrit"
)

func Version(ctx context.Context) string {
	var version string
	// Default tag for previews and PRs.
	if lastHash := lastHash(ctx); lastHash == "" {
		version = time.Now().Format("20060102") + "." + os.Getenv("BUILD_NUMBER")
	} else {
		version = lastHash
	}

	if os.Getenv("BUILD_CAUSE") == "SCMTRIGGER" {
		version = time.Now().Format("20060102") + "." + os.Getenv("BUILD_NUMBER") + ".preview"
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
	if !IsGitHubActions() {
		return !gerrit.IsPreview()
	}
	return os.Getenv("GITHUB_REF_TYPE") == "tag" || os.Getenv("GITHUB_EVENT_NAME") == "workflow_dispatch"
}

func IsGitHubActions() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}

func lastHash(ctx context.Context) string {
	command := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	hash := &bytes.Buffer{}
	command.Stdout = hash
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return ""
	}
	return hash.String()[0:7]
}
