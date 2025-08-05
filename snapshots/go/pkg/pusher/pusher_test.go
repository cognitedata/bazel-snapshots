package pusher

import (
	"crypto/rand"
	"testing"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPush(t *testing.T) {
	store, err := storage.NewStorage("file://" + t.TempDir())
	require.NoError(t, err)

	name := rand.Text()

	pusher := NewPusher(store)
	md, err := pusher.Push(t.Context(), &PushArgs{
		Name: name,
		Snapshot: &models.Snapshot{
			Labels: map[string]*models.Tracker{
				"//path/to:tracker": {
					Digest: "1234abc",
					Run: []string{
						"//path/to:release",
					},
				},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, md.Path, "snapshots/"+name+".json")

	body, err := store.ReadAll(t.Context(), md.Path)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"labels": {
			"//path/to:tracker": {
				"digest": "1234abc",
				"run": [
					"//path/to:release"
				]
			}
		}
	}`, string(body))
}
