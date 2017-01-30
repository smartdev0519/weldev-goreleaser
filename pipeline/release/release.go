package release

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/go-github/github"
	"github.com/goreleaser/goreleaser/clients"
	"github.com/goreleaser/goreleaser/context"
	"golang.org/x/sync/errgroup"
)

// Pipe for github release
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Releasing to GitHub"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	client := clients.GitHub(ctx.Token)

	r, err := getOrCreateRelease(client, ctx)
	if err != nil {
		return err
	}
	var g errgroup.Group
	for _, archive := range ctx.Archives {
		archive := archive
		g.Go(func() error {
			return upload(ctx, client, *r.ID, archive, ctx.Config.Archive.Format)
		})
		for _, format := range ctx.Config.FPM.Formats {
			format := format
			g.Go(func() error {
				return upload(ctx, client, *r.ID, archive, format)
			})
		}
	}
	return g.Wait()
}

func getOrCreateRelease(client *github.Client, ctx *context.Context) (*github.RepositoryRelease, error) {
	owner := ctx.ReleaseRepo.Owner
	repo := ctx.ReleaseRepo.Name
	data := &github.RepositoryRelease{
		Name:    github.String(ctx.Git.CurrentTag),
		TagName: github.String(ctx.Git.CurrentTag),
		Body:    github.String(description(ctx.Git.Diff)),
	}
	r, _, err := client.Repositories.GetReleaseByTag(owner, repo, ctx.Git.CurrentTag)
	if err != nil {
		log.Println("Creating release", ctx.Git.CurrentTag, "on", ctx.Config.Release.Repo)
		r, _, err = client.Repositories.CreateRelease(owner, repo, data)
		return r, err
	}
	log.Println("Updating existing release", ctx.Git.CurrentTag, "on", ctx.Config.Release.Repo)
	r, _, err = client.Repositories.EditRelease(owner, repo, *r.ID, data)
	return r, err
}

func description(diff string) string {
	result := "## Changelog\n" + diff + "\n\n--\nAutomated with @goreleaser"
	cmd := exec.Command("go", "version")
	bts, err := cmd.CombinedOutput()
	if err != nil {
		return result
	}
	return result + "\nBuilt with " + string(bts)
}

func upload(ctx *context.Context, client *github.Client, releaseID int, archive, format string) error {
	archive = archive + "." + format
	var path = filepath.Join("dist", archive)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	log.Println("Uploading", file.Name())
	_, _, err = client.Repositories.UploadReleaseAsset(
		ctx.ReleaseRepo.Owner,
		ctx.ReleaseRepo.Name,
		releaseID,
		&github.UploadOptions{Name: archive},
		file,
	)
	return err
}
