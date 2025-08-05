package getter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/storage"
)

type getter struct{}

func NewGetter() *getter {
	return &getter{}
}

type GetArgs struct {
	Name       string
	StorageUrl string
	SkipNames  bool
	SkipTags   bool
}

func (g *getter) Get(ctx context.Context, args *GetArgs) (*models.Snapshot, error) {
	store, err := storage.NewStorage(args.StorageUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	var snapshotName string
	if !args.SkipTags {
		tagPath := fmt.Sprintf("tags/%s", args.Name)

		snapshotBytes, err := store.ReadAll(ctx, tagPath)
		if err != nil {
			if !errors.Is(err, storage.ErrNotExist) {
				return nil, fmt.Errorf("read tag %q: %w", args.Name, err)
			}
		} else {
			snapshotName = string(snapshotBytes)
		}
	}
	if !args.SkipNames && snapshotName == "" {
		prefix := fmt.Sprintf("snapshots/%s", args.Name)
		for obj, err := range store.List(ctx, prefix) {
			if err != nil {
				return nil, fmt.Errorf("failed to list snapshots with prefix %s: %w", prefix, err)
			}

			// If we've already found a snapshot name
			// and there are still more objects with the same prefix,
			// the name is ambiguous.
			if snapshotName != "" {
				return nil, fmt.Errorf("ambiguous snapshot name: %s", args.Name)
			}

			snapshotName = strings.TrimSuffix(path.Base(obj.Path), ".json")
		}
	}

	snapshotBuffer := new(bytes.Buffer)
	_, err = store.ReadInto(ctx, fmt.Sprintf("snapshots/%s.json", snapshotName), snapshotBuffer)
	if err != nil {
		return nil, fmt.Errorf("failed to find resolved snapshot %s: %w", snapshotName, err)
	}

	snapshot := &models.Snapshot{}
	if err := json.Unmarshal(snapshotBuffer.Bytes(), snapshot); err != nil {
		return nil, fmt.Errorf("snapshot format is invalid: %w", err)
	}

	return snapshot, nil
}
