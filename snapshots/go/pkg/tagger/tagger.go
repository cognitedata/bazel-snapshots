package tagger

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strings"

	"go.beyondstorage.io/v5/types"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/storage"
)

type tagger struct{}

func NewTagger() *tagger {
	return &tagger{}
}

func (*tagger) Tag(ctx context.Context, storageUrl, snapshotName, tagName string) (*types.Object, error) {
	store, err := storage.NewStorage(storageUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	snapshotLocation := fmt.Sprintf("snapshots/%s.json", snapshotName)

	attrs, err := store.StatWithContext(ctx, snapshotLocation)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	tagContent := []byte(strings.TrimSuffix(path.Base(attrs.Path), ".json"))
	tagLocation := fmt.Sprintf("tags/%s", tagName)
	reader := bytes.NewReader(tagContent)

	if _, err := store.WriteWithContext(ctx, tagLocation, reader, int64(reader.Len())); err != nil {
		return nil, fmt.Errorf("failed to write tag: %w", err)
	}

	obj, err := store.StatWithContext(ctx, tagLocation)
	if err != nil {
		return nil, fmt.Errorf("failed to get object details: %w", err)
	}

	return obj, nil
}
