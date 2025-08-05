package pusher

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/storage"
)

type pusher struct{}

func NewPusher() *pusher {
	return &pusher{}
}

type PushArgs struct {
	Name       string
	StorageUrl string
	Snapshot   *models.Snapshot
}

func (p *pusher) Push(ctx context.Context, args *PushArgs) (*storage.ObjectMetadata, error) {
	if args.Snapshot == nil {
		return nil, fmt.Errorf("no snapshot specified")
	}

	snapshotBytes, err := json.MarshalIndent(args.Snapshot, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	store, err := storage.NewStorage(args.StorageUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	location := fmt.Sprintf("snapshots/%s.json", args.Name)
	if err := store.WriteAll(ctx, location, snapshotBytes); err != nil {
		return nil, fmt.Errorf("failed to write to bucket file: %w", err)
	}

	obj, err := store.Stat(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to get object details: %w", err)
	}
	return obj, nil
}
