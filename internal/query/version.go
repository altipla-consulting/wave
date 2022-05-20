package query

import (
	"encoding/base64"
	"os"
	"path"
	"time"

	"libs.altipla.consulting/errors"
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

func GerritChangeNumber() string {
	return os.Getenv("GERRIT_CHANGE_NUMBER")
}

func GerritPatchSet() string {
	return os.Getenv("GERRIT_PATCHSET_NUMBER")
}

func GerritDescriptor() string {
	if os.Getenv("BUILD_CAUSE") == "SCMTRIGGER" {
		return GerritChangeNumber() + "-" + GerritPatchSet()
	}
	return "master"
}

func GerritCommitHash() string {
	return os.Getenv("GERRIT_PATCHSET_REVISION")
}

func GerritCommitMessage() (string, error) {
	msg, err := base64.StdEncoding.DecodeString(os.Getenv("GERRIT_CHANGE_COMMIT_MESSAGE"))
	return string(msg), errors.Trace(err)
}
