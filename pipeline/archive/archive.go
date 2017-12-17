// Package archive implements the pipe interface with the intent of
// archiving and compressing the binaries, readme, and other artifacts. It
// also provides an Archive interface which represents an archiving format.
package archive

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/mattn/go-zglob"
	"golang.org/x/sync/errgroup"

	"github.com/goreleaser/archive"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/archiveformat"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/nametemplate"
)

// Pipe for archive
type Pipe struct{}

func (Pipe) String() string {
	return "creating archives"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	var g errgroup.Group
	var filtered = ctx.Artifacts.Filter(artifact.ByType(artifact.Binary))
	for _, artifacts := range filtered.GroupByPlatform() {
		artifacts := artifacts
		g.Go(func() error {
			if ctx.Config.Archive.Format == "binary" {
				return skip(ctx, artifacts)
			}
			return create(ctx, artifacts)
		})
	}
	return g.Wait()
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Archive.NameTemplate == "" {
		ctx.Config.Archive.NameTemplate = "{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}"
	}
	if ctx.Config.Archive.Format == "" {
		ctx.Config.Archive.Format = "tar.gz"
	}
	if len(ctx.Config.Archive.Files) == 0 {
		ctx.Config.Archive.Files = []string{
			"licence*",
			"LICENCE*",
			"license*",
			"LICENSE*",
			"readme*",
			"README*",
			"changelog*",
			"CHANGELOG*",
		}
	}
	return nil
}

func create(ctx *context.Context, artifacts []artifact.Artifact) error {
	var format = archiveformat.For(ctx, artifacts[0].Platform())
	folder, err := nametemplate.Apply(ctx, artifacts[0])
	if err != nil {
		return err
	}
	archivePath := filepath.Join(ctx.Config.Dist, folder+"."+format)
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %s", archivePath, err.Error())
	}
	defer func() {
		if e := archiveFile.Close(); e != nil {
			log.WithField("archive", archivePath).Errorf("failed to close file: %v", e)
		}
	}()
	log.WithField("archive", archivePath).Info("creating")
	var a = archive.New(archiveFile)
	defer func() {
		if e := a.Close(); e != nil {
			log.WithField("archive", archivePath).Errorf("failed to close archive: %v", e)
		}
	}()

	files, err := findFiles(ctx)
	if err != nil {
		return fmt.Errorf("failed to find files to archive: %s", err.Error())
	}
	for _, f := range files {
		if err = a.Add(wrap(ctx, f, folder), f); err != nil {
			return fmt.Errorf("failed to add %s to the archive: %s", f, err.Error())
		}
	}
	for _, binary := range artifacts {
		if err := a.Add(wrap(ctx, binary.Name, folder), binary.Path); err != nil {
			return fmt.Errorf("failed to add %s -> %s to the archive: %s", binary.Path, binary.Name, err.Error())
		}
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Type:   artifact.UploadableArchive,
		Name:   folder + "." + format,
		Path:   archivePath,
		Goos:   artifacts[0].Goos,
		Goarch: artifacts[0].Goarch,
		Goarm:  artifacts[0].Goarm,
	})
	return nil
}

func skip(ctx *context.Context, artifacts []artifact.Artifact) error {
	for _, a := range artifacts {
		log.WithField("binary", a.Name).Info("skip archiving")
		a.Type = artifact.UploadableBinary
		ctx.Artifacts.Add(a)
	}
	return nil
}

func findFiles(ctx *context.Context) (result []string, err error) {
	for _, glob := range ctx.Config.Archive.Files {
		files, err := zglob.Glob(glob)
		if err != nil {
			return result, fmt.Errorf("globbing failed for pattern %s: %s", glob, err.Error())
		}
		result = append(result, files...)
	}
	return
}

// Wrap archive files with folder if set in config.
func wrap(ctx *context.Context, name, folder string) string {
	if ctx.Config.Archive.WrapInDirectory {
		return filepath.Join(folder, name)
	}
	return name
}
