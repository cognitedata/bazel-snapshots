package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransformURL(t *testing.T) {
	// Note: "%2F" == urlencode("/")

	tests := []struct {
		name string
		give string
		want string
	}{
		// Subdirectory paths are converted to ?prefix= query parameters.
		{
			name: "gs with subdir",
			give: "gs://bucket-name/subdir",
			want: "gs://bucket-name?prefix=subdir%2F",
		},
		{
			name: "gs with nested subdir",
			give: "gs://bucket-name/subdir/nested",
			want: "gs://bucket-name?prefix=subdir%2Fnested%2F",
		},
		{
			name: "gs with trailing slash",
			give: "gs://bucket-name/subdir/",
			want: "gs://bucket-name?prefix=subdir%2F",
		},
		{
			name: "s3 with subdir",
			give: "s3://bucket-name/subdir",
			want: "s3://bucket-name?prefix=subdir%2F",
		},

		// No subdir means no prefix.
		{
			name: "gs without subdir",
			give: "gs://bucket-name",
			want: "gs://bucket-name",
		},
		{
			name: "s3 without subdir",
			give: "s3://bucket-name",
			want: "s3://bucket-name",
		},

		// Existing query parameters are preserved.
		{
			name: "s3 with subdir and query params",
			give: "s3://bucket-name/subdir?region=us-west-2",
			want: "s3://bucket-name?prefix=subdir%2F&region=us-west-2",
		},

		// gcs:// is converted to gs://.
		{
			name: "gcs to gs conversion",
			give: "gcs://bucket-name/subdir",
			want: "gs://bucket-name?prefix=subdir%2F",
		},

		// file:// URLs are not modified.
		{
			name: "file URL unchanged",
			give: "file:///path/to/dir",
			want: "file:///path/to/dir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := transformURL(tt.give)
			assert.Equal(t, tt.want, got)
		})
	}
}
