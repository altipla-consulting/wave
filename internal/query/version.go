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
	// Custom override.
	if ref := os.Getenv("WAVE_VERSION"); ref != "" {
		return ref
	}

	// Gerrit tags.
	if ref := os.Getenv("GERRIT_REFNAME"); ref != "" && ref != "master" {
		return path.Base(ref)
	}

	// GitHub releases.
	if ref := os.Getenv("GITHUB_REF"); ref != "" {
		return path.Base(ref)
	}

	// Default tag for previews and PRs.
	if os.Getenv("BUILD_NUMBER") != "" {
		version := time.Now().Format("20060102") + "." + os.Getenv("BUILD_NUMBER")
		if os.Getenv("BUILD_CAUSE") == "SCMTRIGGER" {
			version += ".preview"
		}
		return version
	}

	// Last strategy is to use the last commit hash.
	return lastHash(ctx)
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
	var hash bytes.Buffer
	command.Stdout = &hash
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return ""
	}
	return hash.String()[0:7]
}
