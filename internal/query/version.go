package query

import (
	"os"
	"path"
	"time"
)

func Version(forceProduction bool) string {
	version := time.Now().Format("20060102") + "." + os.Getenv("BUILD_NUMBER")
	if ref := os.Getenv("GITHUB_REF"); ref != "" {
		version = path.Base(ref)
	}
	if os.Getenv("BUILD_CAUSE") == "SCMTRIGGER" && !forceProduction {
		version += ".preview"
	}

	return version
}
