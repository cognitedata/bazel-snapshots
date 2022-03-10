/* Copyright 2022 Cognite AS */

package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	flag "github.com/spf13/pflag"
	"go.beyondstorage.io/v5/types"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/config"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/storage"
)

type tagConfig struct {
	commonConfig
	bazelConfig  // needed for workspace path
	snapshotName string
	tagName      string
}

const tagName = "_tag"

func getTagConfig(c *config.Config) *tagConfig {
	tc := c.Exts[tagName].(*tagConfig)
	tc.bazelConfig = *getBazelConfig(c)
	tc.commonConfig = *getCommonConfig(c)
	return tc
}

type tagConfigurer struct{}

func (*tagConfigurer) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	tc := &tagConfig{}
	c.Exts[tagName] = tc
	fs.StringVar(&tc.snapshotName, "name", "", "name of snapshot to tag (defaults to HEAD git sha)")
}

func (*tagConfigurer) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	tc := getTagConfig(c)

	// if name is not set, find name from git head
	if tc.snapshotName == "" {
		head, err := getGitHead(tc.workspacePath)
		if err != nil {
			return fmt.Errorf("failed to find name from git: %w", err)
		}
		tc.snapshotName = head
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("need one argument for the tag name, got: %s", fs.Args())
	}
	tc.tagName = fs.Arg(0)

	return nil
}

func runTag(args []string) error {
	cexts := []config.Configurer{
		&bazelConfigurer{},
		&tagConfigurer{},
	}
	c, err := newConfiguration("tag", args, cexts, tagUsage)
	if err != nil {
		return err
	}

	ctx := context.Background()

	tc := getTagConfig(c)

	log.Printf("workspace: %s", tc.workspacePath)
	log.Printf("bucket:    %s", tc.gcsBucket)
	log.Printf("snapshot:  %s", tc.snapshotName)
	log.Printf("tag:       %s", tc.tagName)

	obj, err := tag(ctx, tc)
	if err != nil {
		return err
	}

	log.Printf("tagged snapshot %s as %s: %s", tc.snapshotName, tc.tagName, obj.Path)

	return nil
}

func tag(ctx context.Context, tc *tagConfig) (*types.Object, error) {
	store, err := storage.NewStorage(tc.storageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	snapshotLocation := fmt.Sprintf("%s/snapshots/%s.json", tc.workspaceName, tc.snapshotName)

	attrs, err := store.StatWithContext(ctx, snapshotLocation)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	tagContent := []byte(strings.TrimSuffix(path.Base(attrs.Path), ".json"))
	tagLocation := fmt.Sprintf("%s/tags/%s", tc.workspaceName, tc.tagName)
	reader := bytes.NewReader(tagContent)

	if _, err := store.WriteWithContext(ctx, tagLocation, reader, int64(reader.Len())); err != nil {
		return nil, fmt.Errorf("failed to write tag: %w", err)
	}

	obj, err := store.StatWithContext(ctx, tagLocation)
	if err != nil {
		return nil, fmt.Errorf("failed to get object details: %w", err)
	}

	return obj, nil
}

func tagUsage(fs *flag.FlagSet) {
	fmt.Fprint(os.Stderr, `usage: tag --name <snapshot> <tag>

Assigns a tag to some (pushed) snapshot, referenced by name. Snapshot name
defaults to the current git HEAD. Tagging a snapshot creates a named
reference to it. For example, a tag "deployed" can be a reference to the
snapshot which was most recently deployed.

Example:
	snapshot tag --name <some-snapshot> mytag

FLAGS:
`)
	fs.PrintDefaults()
}
