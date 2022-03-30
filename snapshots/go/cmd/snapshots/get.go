/* Copyright 2022 Cognite AS */

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

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
	store, err := storage.NewStorage(gc.storageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	snapshotBuffer := new(bytes.Buffer)
	var snapshotName string

	if !gc.skipTags {
		tagPath := fmt.Sprintf("tags/%s", gc.name)
		tagBuffer := new(bytes.Buffer)
		_, err := store.StatWithContext(ctx, tagPath)
		if err == nil {
			_, err = store.ReadWithContext(ctx, tagPath, tagBuffer)
			if err != nil {
				return nil, fmt.Errorf("failed to look for tag %s: %w", gc.name, err)
			}
			snapshotBytes, err := ioutil.ReadAll(tagBuffer)
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

	if !gc.skipNames && snapshotBuffer.Len() == 0 {
		it, err := store.List(fmt.Sprintf("snapshots/%s", gc.name))
		if err != nil {
			return nil, fmt.Errorf("cannot create object iterator: %w", err)
		}
		if attrs, err := it.Next(); err != nil && errors.Is(err, storage.IteratorDone) {
			return nil, fmt.Errorf("failed to look for snapshot %s in %s", gc.name, store.String())
		} else if err == nil {
			if _, err := it.Next(); err == nil {
				return nil, fmt.Errorf("ambiguous snapshot name: %s", gc.name)
			}
			snapshotName = strings.TrimSuffix(path.Base(attrs.Path), ".json")
		}

		_, err = store.ReadWithContext(ctx, fmt.Sprintf("snapshots/%s.json", snapshotName), snapshotBuffer)
		if err != nil {
			return nil, fmt.Errorf("cannot read the snapshot: %w", err)
		}
	}

	if snapshotBuffer.Len() == 0 {
		return nil, fmt.Errorf("could not find tag or snapshot: %s", gc.name)
	}

	snapshot := &models.Snapshot{}
	if err := json.Unmarshal(snapshotBuffer.Bytes(), snapshot); err != nil {
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
