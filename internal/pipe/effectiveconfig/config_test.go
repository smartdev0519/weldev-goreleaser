package effectiveconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestPipeDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestRun(t *testing.T) {
	var folder = testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	var ctx = context.New(
		config.Project{
			Dist: dist,
		},
	)
	require.NoError(t, Pipe{}.Run(ctx))
	bts, err := os.ReadFile(filepath.Join(dist, "config.yaml"))
	require.NoError(t, err)
	require.NotEmpty(t, string(bts))
}
