/* Copyright 2022 Cognite AS */

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	flag "github.com/spf13/pflag"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/config"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/getter"
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

	snapshot, err := getter.NewGetter().Get(ctx, gc.name, gc.storageURL, gc.skipNames, gc.skipTags)
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
