package query

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path"
	"strings"

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
	if ref := os.Getenv("GITHUB_REF"); ref != "" && strings.HasPrefix(ref, "refs/tags/") {
		return path.Base(ref)
	}

	// Jenkins previews.
	if os.Getenv("BUILD_NUMBER") != "" {
		if gerrit.IsPreview() {
			return gerrit.SimulatedBranch()
		}
		return os.Getenv("BUILD_ID") + "-" + os.Getenv("GERRIT_NEWREV")
	}

	// Last strategy is to use the last commit hash.
	return lastHash(ctx)
}

func VersionHostname(override string) string {
	if override != "" {
		return override
	}
	if os.Getenv("GERRIT_EVENT_TYPE") == "patchset-created" {
		return gerrit.SimulatedBranch()
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
