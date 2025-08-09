package pusher

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/storage"
)

type Storage interface {
	WriteAll(ctx context.Context, location string, data []byte) error
	Stat(ctx context.Context, location string) (*storage.ObjectMetadata, error)
}

var _ Storage = (*storage.Storage)(nil)

type pusher struct {
	store Storage
}

func NewPusher(store Storage) *pusher {
	return &pusher{store: store}
}

type PushArgs struct {
	Name     string
	Snapshot *models.Snapshot
}

func (p *pusher) Push(ctx context.Context, args *PushArgs) (*storage.ObjectMetadata, error) {
	if args.Snapshot == nil {
		return nil, fmt.Errorf("no snapshot specified")
	}

	snapshotBytes, err := json.MarshalIndent(args.Snapshot, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	location := fmt.Sprintf("snapshots/%s.json", args.Name)
	if err := p.store.WriteAll(ctx, location, snapshotBytes); err != nil {
		return nil, fmt.Errorf("failed to write to bucket file: %w", err)
	}

	obj, err := p.store.Stat(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to get object details: %w", err)
	}
	return obj, nil
}
