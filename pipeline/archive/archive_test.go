package archive

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestRunPipe(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, os.Mkdir(filepath.Join(dist, "mybin_darwin_amd64"), 0755))
	assert.NoError(t, os.Mkdir(filepath.Join(dist, "mybin_windows_amd64"), 0755))
	_, err := os.Create(filepath.Join(dist, "mybin_darwin_amd64", "mybin"))
	assert.NoError(t, err)
	_, err = os.Create(filepath.Join(dist, "mybin_windows_amd64", "mybin.exe"))
	assert.NoError(t, err)
	_, err = os.Create(filepath.Join(folder, "README.md"))
	assert.NoError(t, err)
	var ctx = &context.Context{
		Config: config.Project{
			Dist: dist,
			Archive: config.Archive{
				Files: []string{
					"README.*",
				},
				FormatOverrides: []config.FormatOverride{
					{
						Goos:   "windows",
						Format: "zip",
					},
				},
			},
		},
	}
	ctx.AddBinary("darwinamd64", "mybin_darwin_amd64", "mybin", filepath.Join(dist, "mybin_darwin_amd64", "mybin"))
	ctx.AddBinary("windowsamd64", "mybin_windows_amd64", "mybin.exe", filepath.Join(dist, "mybin_windows_amd64", "mybin.exe"))
	for _, format := range []string{"tar.gz", "zip"} {
		t.Run("Archive format "+format, func(t *testing.T) {
			ctx.Config.Archive.Format = format
			assert.NoError(t, Pipe{}.Run(ctx))
		})
	}
}

func TestRunPipeBinary(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, os.Mkdir(filepath.Join(dist, "mybin_darwin"), 0755))
	assert.NoError(t, os.Mkdir(filepath.Join(dist, "mybin_win"), 0755))
	_, err := os.Create(filepath.Join(dist, "mybin_darwin", "mybin"))
	assert.NoError(t, err)
	_, err = os.Create(filepath.Join(dist, "mybin_win", "mybin.exe"))
	assert.NoError(t, err)
	_, err = os.Create(filepath.Join(folder, "README.md"))
	assert.NoError(t, err)
	var ctx = &context.Context{
		Config: config.Project{
			Dist: dist,
			Builds: []config.Build{
				{Binary: "mybin"},
			},
			Archive: config.Archive{
				Format: "binary",
			},
		},
	}
	ctx.AddBinary("darwinamd64", "mybin_darwin", "mybin", filepath.Join(dist, "mybin_darwin", "mybin"))
	ctx.AddBinary("windowsamd64", "mybin_win", "mybin.exe", filepath.Join(dist, "mybin_win", "mybin.exe"))
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Contains(t, ctx.Artifacts, "mybin_darwin/mybin")
	assert.Contains(t, ctx.Artifacts, "mybin_win/mybin.exe")
	assert.Len(t, ctx.Artifacts, 2)
}

func TestRunPipeDistRemoved(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Dist: "/path/nope",
			Archive: config.Archive{
				Format: "zip",
			},
		},
	}
	ctx.AddBinary("windowsamd64", "nope", "no", "blah")
	assert.Error(t, Pipe{}.Run(ctx))
}

func TestRunPipeInvalidGlob(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Dist: "/tmp",
			Archive: config.Archive{
				Files: []string{
					"[x-]",
				},
			},
		},
	}
	ctx.AddBinary("windowsamd64", "whatever", "foo", "bar")
	assert.Error(t, Pipe{}.Run(ctx))
}

func TestRunPipeGlobFailsToAdd(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	assert.NoError(t, os.MkdirAll(filepath.Join(folder, "folder", "another"), 0755))

	var ctx = &context.Context{
		Config: config.Project{
			Dist: folder,
			Archive: config.Archive{
				Files: []string{
					"folder",
				},
			},
		},
	}
	ctx.AddBinary("windows386", "mybin", "mybin", "dist/mybin")
	assert.Error(t, Pipe{}.Run(ctx))
}
