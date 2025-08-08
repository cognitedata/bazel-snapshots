package tagger

import (
	"testing"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTag(t *testing.T) {
	store, err := storage.NewStorage("file://" + t.TempDir())
	require.NoError(t, err)

	require.NoError(t, store.WriteAll(
		t.Context(), "snapshots/foo.json", []byte(`{"test":"data"}`)))

	tagger := NewTagger(store)
	md, err := tagger.Tag(t.Context(), &TagArgs{
		SnapshotName: "foo",
		TagName:      "latest",
	})
	require.NoError(t, err)

	assert.Equal(t, "tags/latest", md.Path)

	gotTag, err := store.ReadAll(t.Context(), "tags/latest")
	require.NoError(t, err)
	assert.Equal(t, "foo", string(gotTag))
}
