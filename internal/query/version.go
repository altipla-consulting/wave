package query

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path"
	"strings"
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
	if ref := os.Getenv("GITHUB_REF"); ref != "" && strings.HasPrefix(ref, "refs/tags/") {
		return path.Base(ref)
	}

	// Jenkins previews.
	if os.Getenv("BUILD_NUMBER") != "" {
		if gerrit.IsPreview() {
			return time.Now().Format("20060102") + "." + os.Getenv("BUILD_NUMBER") + ".0-preview." + gerrit.ChangeNumber() + "." + gerrit.PatchSet()
		}
		return time.Now().Format("20060102") + "." + os.Getenv("BUILD_NUMBER") + ".0+" + os.Getenv("GERRIT_NEWREV")[0:7]
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

func VersionImageTag(ctx context.Context) string {
	version := Version(ctx)
	version = strings.Replace(version, "+", "-", 1)
	return version
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
