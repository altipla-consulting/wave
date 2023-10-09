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
	lastHash := GetLastHash(ctx)
	if lastHash == "" {
		version = time.Now().Format("20060102") + "." + os.Getenv("BUILD_NUMBER")
	}
	version = lastHash[:8]

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
	if !IsGitHubActions() {
		return !gerrit.IsPreview()
	}
	return os.Getenv("GITHUB_REF_TYPE") == "tag" || os.Getenv("GITHUB_EVENT_NAME") == "workflow_dispatch"
}

func IsGitHubActions() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}

func GetLastHash(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	bytes := &bytes.Buffer{}
	cmd.Stdout = bytes
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return ""
	}
	return bytes.String()
}
