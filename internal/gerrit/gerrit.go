package gerrit

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"libs.altipla.consulting/errors"
)

func ChangeNumber() string {
	return os.Getenv("GERRIT_CHANGE_NUMBER")
}

func PatchSet() string {
	return os.Getenv("GERRIT_PATCHSET_NUMBER")
}

func Descriptor() string {
	if os.Getenv("BUILD_CAUSE") == "SCMTRIGGER" {
		return "preview-" + ChangeNumber() + "-" + PatchSet()
	}
	return "master"
}

func IsPreview() bool {
	return Descriptor() != "master"
}

func CommitHash() string {
	return os.Getenv("GERRIT_PATCHSET_REVISION")
}

func CommitMessage() (string, error) {
	msg, err := base64.StdEncoding.DecodeString(os.Getenv("GERRIT_CHANGE_COMMIT_MESSAGE"))
	return string(msg), errors.Trace(err)
}

func Host() string {
	return os.Getenv("GERRIT_HOST")
}

func Port() string {
	return os.Getenv("GERRIT_PORT")
}

func BotUsername() string {
	return os.Getenv("GERRIT_BOT_USERNAME")
}

func Comment(msg string) error {
	ssh := []string{
		"ssh",
		"-p", Port(),
		fmt.Sprintf("%s@%s", BotUsername(), Host()),
		"gerrit", "review", fmt.Sprintf("%v,%v", ChangeNumber(), PatchSet()),
		"--message", strconv.Quote(msg),
	}
	log.Debug(strings.Join(ssh, " "))
	comment := exec.Command(ssh[0], ssh[1:]...)
	comment.Stdout = os.Stdout
	comment.Stderr = os.Stderr
	return errors.Trace(comment.Run())
}
