package tagger

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/storage"
)

type Storage interface {
	WriteAll(ctx context.Context, location string, data []byte) error
	Stat(ctx context.Context, location string) (*storage.ObjectMetadata, error)
}

var _ Storage = (*storage.Storage)(nil)

type tagger struct {
	store Storage
}

func NewTagger(store Storage) *tagger {
	return &tagger{store: store}
}

type TagArgs struct {
	SnapshotName string
	TagName      string
}

func (t *tagger) Tag(ctx context.Context, args *TagArgs) (*storage.ObjectMetadata, error) {
	snapshotLocation := fmt.Sprintf("snapshots/%s.json", args.SnapshotName)

	attrs, err := t.store.Stat(ctx, snapshotLocation)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	tagContent := []byte(strings.TrimSuffix(path.Base(attrs.Path), ".json"))
	tagLocation := fmt.Sprintf("tags/%s", args.TagName)
	if err := t.store.WriteAll(ctx, tagLocation, tagContent); err != nil {
		return nil, fmt.Errorf("failed to write tag: %w", err)
	}

	obj, err := t.store.Stat(ctx, tagLocation)
	if err != nil {
		return nil, fmt.Errorf("failed to get object details: %w", err)
	}

	return obj, nil
}
