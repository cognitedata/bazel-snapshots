/* Copyright 2022 Cognite AS */

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/getter"
)

type getCmd struct {
	skipTags  bool
	skipNames bool
	name      string

	storageURL string

	cmd *cobra.Command
}

func newGetCmd() *getCmd {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get snapshot",
		Long: `Resolves and fetches a snapshot, either by tag or snapshot name. Tag has
priority.`,
		Args: cobra.ExactArgs(1),
	}

	gc := &getCmd{
		cmd: cmd,
	}

	cmd.PersistentFlags().BoolVar(&gc.skipTags, "skip-tags", false, "don't look up by tag")
	cmd.PersistentFlags().BoolVar(&gc.skipNames, "skip-names", false, "don't look up by name")

	cmd.RunE = gc.runGet

	return gc
}

func (gc *getCmd) checkArgs(args []string) error {
	gc.name = args[0]

	storageURL, err := gc.cmd.Flags().GetString("storage-url")
	if err != nil {
		return err
	}
	gc.storageURL = storageURL

	return nil
}

func (gc *getCmd) runGet(cmd *cobra.Command, args []string) error {
	err := gc.checkArgs(args)
	if err != nil {
		return err
	}

	ctx := context.Background()

	getArgs := getter.GetArgs{
		Name:       gc.name,
		StorageURL: gc.storageURL,
		SkipNames:  gc.skipNames,
		SkipTags:   gc.skipTags,
	}
	snapshot, err := getter.NewGetter().Get(ctx, &getArgs)
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
