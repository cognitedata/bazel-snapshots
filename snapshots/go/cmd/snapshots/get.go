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

	storageUrl string

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

	cmd.PersistentFlags().BoolVar(&gc.skipTags, "skip-tags", false, "Don't look up by tag")
	cmd.PersistentFlags().BoolVar(&gc.skipNames, "skip-names", false, "Don't look up by name")

	cmd.RunE = gc.runGet

	return gc
}

func (gc *getCmd) checkArgs(args []string) error {
	gc.name = args[0]

	storageUrl, err := gc.cmd.Flags().GetString("storage-url")
	if err != nil {
		return err
	}
	gc.storageUrl = storageUrl

	return nil
}

func (gc *getCmd) runGet(cmd *cobra.Command, args []string) error {
	err := gc.checkArgs(args)
	if err != nil {
		return err
	}

	ctx := context.Background()
	snapshot, err := getter.NewGetter().Get(ctx, gc.name, gc.storageUrl, gc.skipNames, gc.skipTags)
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
