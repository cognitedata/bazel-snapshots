package getter

import (
	"testing"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	store, err := storage.NewStorage("file://" + t.TempDir())
	require.NoError(t, err)

	files := map[string]string{
		"snapshots/abc123.json": `{"labels": {"//foo": {"digest": "abc123", "run": ["//foo:deploy"]}}}`,
		"snapshots/abc456.json": `{"labels": {"//foo": {"digest": "abc456", "run": ["//foo:deploy"]}}}`,
		"tags/latest":           "abc456",
		"tags/broken":           "nonexistent",
	}
	for file, content := range files {
		err := store.WriteAll(t.Context(), file, []byte(content))
		require.NoError(t, err)
	}

	getter := NewGetter(store)

	t.Run("ByTag", func(t *testing.T) {
		snapshot, err := getter.Get(t.Context(), &GetArgs{Name: "latest"})
		require.NoError(t, err)
		require.NotNil(t, snapshot)
		require.Equal(t, "abc456", snapshot.Labels["//foo"].Digest)
	})

	t.Run("ByName", func(t *testing.T) {
		snapshot, err := getter.Get(t.Context(), &GetArgs{Name: "abc123", SkipTags: true})
		require.NoError(t, err)
		require.NotNil(t, snapshot)
		require.Equal(t, "abc123", snapshot.Labels["//foo"].Digest)
	})

	t.Run("ByNamePrefix", func(t *testing.T) {
		snapshot, err := getter.Get(t.Context(), &GetArgs{Name: "abc1", SkipTags: true})
		require.NoError(t, err)
		require.NotNil(t, snapshot)
		require.Equal(t, "abc123", snapshot.Labels["//foo"].Digest)
	})

	t.Run("ByNameAmbiguous", func(t *testing.T) {
		_, err := getter.Get(t.Context(), &GetArgs{Name: "abc", SkipTags: true})
		require.Error(t, err)
		assert.ErrorContains(t, err, "ambiguous snapshot name")
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := getter.Get(t.Context(), &GetArgs{Name: "nonexistent"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "not found")
	})

	t.Run("BrokenTag", func(t *testing.T) {
		_, err := getter.Get(t.Context(), &GetArgs{Name: "broken"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "find resolved snapshot")
	})
}
