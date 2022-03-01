/* Copyright 2022 Cognite AS */

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/graymeta/stow"
	flag "github.com/spf13/pflag"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/config"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/storage"
)

type getConfig struct {
	commonConfig
	skipTags  bool // don't try to resolve by tag
	skipNames bool // don't try to resolve by name
	name      string
}

const getName = "_get"

func getGetConfig(c *config.Config) *getConfig {
	gc := c.Exts[getName].(*getConfig)
	gc.commonConfig = *getCommonConfig(c)
	return gc
}

type getConfigurer struct{}

func (*getConfigurer) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	gc := &getConfig{}
	c.Exts[getName] = gc
	fs.BoolVar(&gc.skipTags, "skip-tags", false, "don't look up by tag")
	fs.BoolVar(&gc.skipNames, "skip-names", false, "don't look up by name")
}

func (*getConfigurer) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	if fs.NArg() != 1 {
		return fmt.Errorf("need exactly one argument")
	}

	gc := getGetConfig(c)
	gc.name = fs.Arg(0)

	return nil
}

func runGet(args []string) error {
	cexts := []config.Configurer{
		&getConfigurer{},
	}
	c, err := newConfiguration("get", args, cexts, getUsage)
	if err != nil {
		return err
	}

	ctx := context.Background()

	gc := getGetConfig(c)

	snapshot, err := get(ctx, gc)
	if err != nil {
		return err
	}

	snapshotBytes, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	io.Copy(os.Stdout, bytes.NewReader(snapshotBytes))

	return nil
}

func get(ctx context.Context, gc *getConfig) (*models.Snapshot, error) {
	// We let google to find out credentials and projectID
	cfg := storage.Config("", "")
	loc, err := storage.Dial("google", cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	bucket, err := loc.Container(gc.gcsBucket)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	var snapshotAttrs storage.Item

	if !gc.skipTags {
		tagReader, err := bucket.Item(fmt.Sprintf("%s/tags/%s", gc.workspaceName, gc.name))
		// NOTE(taylan): Old code had err != storage.ErrObjectNotExist check. We might want to
		// Implement that later on.
		if err == nil {
			// Finds the snapshot inside `deployed` file.
			snapshotName, err := storage.ToString(tagReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read tag: %w", err)
			}

			// Fetches the snapshot that we deployed last.
			snapshotAttrs, err = bucket.Item(fmt.Sprintf("%s/snapshots/%s.json", gc.workspaceName, snapshotName))
			if err != nil {
				return nil, fmt.Errorf("failed to find resolved snapshot %s: %w", snapshotName, err)
			}
		}
	}

	if !gc.skipNames && snapshotAttrs == nil {
		prefix := fmt.Sprintf("%s/snapshots/%s", gc.workspaceName, gc.name)
		err := storage.Walk(bucket, prefix, 50, func(item stow.Item, err error) error {
			if err != nil {
				return err
			}
			snapshotAttrs = item
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("Failed to look for snapshot: %w", err)
		}
	}

	if snapshotAttrs == nil {
		return nil, fmt.Errorf("could not find tag or snapshot: %s", gc.name)
	}

	// r, err := bucket.Object(snapshotAttrs.Name).NewReader(ctx)
	i, err := bucket.Item(snapshotAttrs.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to get resolved snapshot %s: %w", snapshotAttrs.Name(), err)
	}

	r, err := i.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open snapshot: %w", err)
	}

	defer r.Close()

	snapshotBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot: %w", err)
	}

	snapshot := &models.Snapshot{}
	if err := json.Unmarshal(snapshotBytes, snapshot); err != nil {
		return nil, fmt.Errorf("snapshot format is invalid: %w", err)
	}

	return snapshot, nil
}

func getUsage(fs *flag.FlagSet) {
	fmt.Fprint(os.Stderr, `usage: get <snapshot/tag>

Resolves and fetches a snapshot, either by tag or snapshot name. Tag has
priority.

Examples:
	snapshots get deployed

FLAGS:
`)
	fs.PrintDefaults()
}
