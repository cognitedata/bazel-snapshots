package storage

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteAllAndReadAll(t *testing.T) {
	storage, err := NewStorage("file://" + t.TempDir())
	require.NoError(t, err)

	contents := []byte("test content")

	require.NoError(t, storage.WriteAll(t.Context(), "test-file.txt", contents))

	got, err := storage.ReadAll(t.Context(), "test-file.txt")
	require.NoError(t, err)

	assert.Equal(t, contents, got)
}

func TestStat(t *testing.T) {
	storage, err := NewStorage("file://" + t.TempDir())
	require.NoError(t, err)

	contents := []byte("test content for stat")

	require.NoError(t, storage.WriteAll(t.Context(), "test-file.txt", contents))

	got, err := storage.Stat(t.Context(), "test-file.txt")
	require.NoError(t, err)

	assert.Equal(t, "test-file.txt", got.Path)
	assert.Equal(t, int64(len(contents)), got.ContentLength)
}

func TestErrNotFound(t *testing.T) {
	storage, err := NewStorage("file://" + t.TempDir())
	require.NoError(t, err)

	t.Run("ReadAll", func(t *testing.T) {
		_, err := storage.ReadAll(t.Context(), "non-existent-file.txt")
		assert.ErrorIs(t, err, ErrNotExist)
	})

	t.Run("Stat", func(t *testing.T) {
		_, err := storage.Stat(t.Context(), "non-existent-file.txt")
		assert.ErrorIs(t, err, ErrNotExist)
	})

	t.Run("ReadInto", func(t *testing.T) {
		var buf bytes.Buffer
		_, err := storage.ReadInto(t.Context(), "non-existent-file.txt", &buf)
		assert.ErrorIs(t, err, ErrNotExist)
	})
}

func TestReadInto(t *testing.T) {
	storage, err := NewStorage("file://" + t.TempDir())
	require.NoError(t, err)

	contents := []byte("test content for read into")

	require.NoError(t, storage.WriteAll(t.Context(), "test-file.txt", contents))

	var buf bytes.Buffer
	got, err := storage.ReadInto(t.Context(), "test-file.txt", &buf)
	require.NoError(t, err)

	assert.Equal(t, int64(len(contents)), got)
	assert.Equal(t, contents, buf.Bytes())
}

func TestList(t *testing.T) {
	storage, err := NewStorage("file://" + t.TempDir())
	require.NoError(t, err)

	files := []string{
		"foobar",
		"foobaz",
		"foo/bar",
		"foo/qux",
		"other/file",
	}

	for _, path := range files {
		require.NoError(t, storage.WriteAll(t.Context(), path, nil))
	}

	t.Run("PrefixFile", func(t *testing.T) {
		var got []string
		for obj, err := range storage.List(t.Context(), "foo") {
			require.NoError(t, err)
			got = append(got, obj.Path)
		}

		assert.ElementsMatch(t, []string{
			"foobar",
			"foobaz",
		}, got)
	})

	t.Run("PrefixDir", func(t *testing.T) {
		var got []string
		for obj, err := range storage.List(t.Context(), "foo/") {
			require.NoError(t, err)
			got = append(got, obj.Path)
		}

		assert.ElementsMatch(t, []string{
			"foo/bar",
			"foo/qux",
		}, got)
	})

	t.Run("PrefixDirFile", func(t *testing.T) {
		var got []string
		for obj, err := range storage.List(t.Context(), "foo/b") {
			require.NoError(t, err)
			got = append(got, obj.Path)
		}

		assert.ElementsMatch(t, []string{
			"foo/bar",
		}, got)
	})
}

// Verifies that s3://bucket/subdir URLs write to the subdir prefix,
// not the bucket root.
func TestS3SubdirectoryURL(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path

		// Return a minimal response for PUT requests.
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Return S3-style NotFound for anything else.
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<Error>
	<Code>NoSuchKey</Code>
	<Message>The specified key does not exist.</Message>
</Error>`)
	}))
	defer srv.Close()

	// use_path_style tells the AWS SDK to put the bucket name in the path,
	// e.g.
	//    example.com/bucket-name/subdir
	// Instead of,
	//    bucket-name.example.com/subdir
	storageURL := fmt.Sprintf("s3://mybucket/subdir?endpoint=%s&use_path_style=true&anonymous=true", srv.URL)

	store, err := NewStorage(storageURL)
	require.NoError(t, err)

	_ = store.WriteAll(t.Context(), "test.txt", []byte("hello"))
	assert.Equal(t, "/mybucket/subdir/test.txt", capturedPath)
}
