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

	"cloud.google.com/go/storage"
	flag "github.com/spf13/pflag"
	"google.golang.org/api/iterator"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/config"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
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
	sclient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	bucket := sclient.Bucket(gc.gcsBucket)

	var snapshotAttrs *storage.ObjectAttrs

	if !gc.skipTags {
		tagReader, err := bucket.Object(fmt.Sprintf("%s/tags/%s", gc.workspaceName, gc.name)).NewReader(ctx)
		if err != nil && err != storage.ErrObjectNotExist {
			return nil, fmt.Errorf("failed to look for tag %s: %w", gc.name, err)
		}
		if err == nil {
			snapshotName, err := ioutil.ReadAll(tagReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read tag: %w", err)
			}

			snapshotAttrs, err = bucket.Object(fmt.Sprintf("%s/snapshots/%s.json", gc.workspaceName, snapshotName)).Attrs(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to find resolved snapshot %s: %w", snapshotName, err)
			}
		}
	}

	if !gc.skipNames && snapshotAttrs == nil {
		it := bucket.Objects(ctx, &storage.Query{
			Prefix: fmt.Sprintf("%s/snapshots/%s", gc.workspaceName, gc.name),
		})
		if attrs, err := it.Next(); err != nil && err != iterator.Done {
			return nil, fmt.Errorf("failed to look for snapshot: %w", err)
		} else if err == nil {
			if _, err := it.Next(); err == nil {
				return nil, fmt.Errorf("ambiguous snapshot name: %s", gc.name)
			}
			snapshotAttrs = attrs
		}
	}

	if snapshotAttrs == nil {
		return nil, fmt.Errorf("could not find tag or snapshot: %s", gc.name)
	}

	r, err := bucket.Object(snapshotAttrs.Name).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get resolved snapshot %s: %w", snapshotAttrs.Name, err)
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
