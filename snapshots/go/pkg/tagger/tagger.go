package tagger

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/storage"
)

type tagger struct{}

func NewTagger() *tagger {
	return &tagger{}
}

type TagArgs struct {
	SnapshotName string
	StorageUrl   string
	TagName      string
}

func (*tagger) Tag(ctx context.Context, args *TagArgs) (*storage.ObjectMetadata, error) {
	store, err := storage.NewStorage(args.StorageUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	snapshotLocation := fmt.Sprintf("snapshots/%s.json", args.SnapshotName)

	attrs, err := store.Stat(ctx, snapshotLocation)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	tagContent := []byte(strings.TrimSuffix(path.Base(attrs.Path), ".json"))
	tagLocation := fmt.Sprintf("tags/%s", args.TagName)
	if err := store.WriteAll(ctx, tagLocation, tagContent); err != nil {
		return nil, fmt.Errorf("failed to write tag: %w", err)
	}

	obj, err := store.Stat(ctx, tagLocation)
	if err != nil {
		return nil, fmt.Errorf("failed to get object details: %w", err)
	}

	return obj, nil
}
