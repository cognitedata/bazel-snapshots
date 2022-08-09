package pusher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"go.beyondstorage.io/v5/types"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/storage"
)

type pusher struct {
}

func NewPusher() *pusher {
	return &pusher{}
}

func (p *pusher) Push(ctx context.Context, name, storageUrl string, snapshot *models.Snapshot) (*types.Object, error) {
	if snapshot == nil {
		return nil, fmt.Errorf("no snapshot specified")
	}

	snapshotBytes, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	store, err := storage.NewStorage(storageUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	location := fmt.Sprintf("snapshots/%s.json", name)
	reader := bytes.NewReader(snapshotBytes)
	if _, err := store.WriteWithContext(ctx, location, reader, int64(reader.Len())); err != nil {
		return nil, fmt.Errorf("failed to write to bucket file: %w", err)
	}

	obj, err := store.StatWithContext(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to get object details: %w", err)
	}
	return obj, nil
}
