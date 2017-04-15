// Package git implements the Pipe interface extracting usefull data from
// git and putting it in the context.
package git

import (
	"os"
	"regexp"
	"strings"

	"github.com/goreleaser/goreleaser/context"
)

// ErrInvalidVersionFormat is return when the version isnt in a valid format
type ErrInvalidVersionFormat struct {
	version string
}

func (e ErrInvalidVersionFormat) Error() string {
	return e.version + " is not in a valid version format"
}

// Pipe for brew deployment
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Getting Git info"
}

// Run the pipe
func (p Pipe) Run(ctx *context.Context) (err error) {
	folder, err := os.Getwd()
	if err != nil {
		return err
	}
	return p.doRun(ctx, folder)
}

func (Pipe) doRun(ctx *context.Context, pwd string) (err error) {
	tag, err := cleanGit(pwd, "describe", "--tags", "--abbrev=0", "--always")
	if err != nil {
		return
	}
	prev, err := previous(pwd, tag)
	if err != nil {
		return
	}

	log, err := git(pwd, "log", "--pretty=oneline", "--abbrev-commit", prev+".."+tag)
	if err != nil {
		return
	}

	ctx.Git = context.GitInfo{
		CurrentTag:  tag,
		PreviousTag: prev,
		Diff:        log,
	}
	// removes usual `v` prefix
	ctx.Version = strings.TrimPrefix(tag, "v")
	if versionErr := isVersionValid(ctx.Version); versionErr != nil {
		return versionErr
	}
	commit, err := cleanGit(pwd, "show", "--format='%H'", "HEAD")
	if err != nil {
		return
	}
	ctx.Git.Commit = commit
	return
}

func previous(pwd, tag string) (previous string, err error) {
	previous, err = cleanGit(pwd, "describe", "--tags", "--abbrev=0", "--always", tag+"^")
	if err != nil {
		previous, err = cleanGit(pwd, "rev-list", "--max-parents=0", "HEAD")
	}
	return
}

func isVersionValid(version string) error {
	matches, err := regexp.MatchString("^[0-9.]+", version)
	if err != nil || !matches {
		return ErrInvalidVersionFormat{version}
	}
	return nil
}
