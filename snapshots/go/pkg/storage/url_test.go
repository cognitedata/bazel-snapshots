package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBackwardsCompatibleStorageURL(t *testing.T) {
	tests := []struct {
		give string
		want string
	}{
		{"gs://bucket-name/subdir", "gs://bucket-name/subdir"},
		{"gcs://bucket-name/subdir", "gs://bucket-name/subdir"},
		{"s3://bucket-name/subdir", "s3://bucket-name/subdir"},
	}

	for _, tt := range tests {
		t.Run(tt.give, func(t *testing.T) {
			got := backwardsCompatibleStorageURL(tt.give)
			assert.Equal(t, tt.want, got)
		})
	}
}
