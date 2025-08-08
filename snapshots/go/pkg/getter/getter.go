package getter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	StorageURL string
	SkipNames  bool
	SkipTags   bool
}

func (g *getter) Get(ctx context.Context, args *GetArgs) (*models.Snapshot, error) {
	store, err := storage.NewStorage(args.StorageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	snapshotBuffer := new(bytes.Buffer)
	var snapshotName string

	if !args.SkipTags {
		tagPath := fmt.Sprintf("tags/%s", args.Name)
		tagBuffer := new(bytes.Buffer)
		_, err := store.StatWithContext(ctx, tagPath)
		if err == nil {
			_, err = store.ReadWithContext(ctx, tagPath, tagBuffer)
			if err != nil {
				return nil, fmt.Errorf("failed to look for tag %s: %w", args.Name, err)
			}
			snapshotBytes, err := io.ReadAll(tagBuffer)
			if err != nil {
				return nil, fmt.Errorf("failed to read tag: %w", err)
			}
			snapshotName = string(snapshotBytes)

			_, err = store.ReadWithContext(ctx, fmt.Sprintf("snapshots/%s.json", snapshotName), snapshotBuffer)
			if err != nil {
				return nil, fmt.Errorf("failed to find resolved snapshot %s: %w", snapshotName, err)
			}
		}
	}

	if !args.SkipNames && snapshotBuffer.Len() == 0 {
		it, err := store.List(fmt.Sprintf("snapshots/%s", args.Name))
		if err != nil {
			return nil, fmt.Errorf("cannot create object iterator: %w", err)
		}
		if attrs, err := it.Next(); err != nil && errors.Is(err, storage.IteratorDone) {
			return nil, fmt.Errorf("failed to look for snapshot %s in %s", args.Name, store.String())
		} else if err == nil {
			if _, err := it.Next(); err == nil {
				return nil, fmt.Errorf("ambiguous snapshot name: %s", args.Name)
			}
			snapshotName = strings.TrimSuffix(path.Base(attrs.Path), ".json")
		}

		_, err = store.ReadWithContext(ctx, fmt.Sprintf("snapshots/%s.json", snapshotName), snapshotBuffer)
		if err != nil {
			return nil, fmt.Errorf("cannot read the snapshot: %w", err)
		}
	}

	if snapshotBuffer.Len() == 0 {
		return nil, fmt.Errorf("could not find tag or snapshot: %s", args.Name)
	}

	snapshot := &models.Snapshot{}
	if err := json.Unmarshal(snapshotBuffer.Bytes(), snapshot); err != nil {
		return nil, fmt.Errorf("snapshot format is invalid: %w", err)
	}

	return snapshot, nil
}
