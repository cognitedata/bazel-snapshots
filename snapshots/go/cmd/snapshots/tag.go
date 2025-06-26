/* Copyright 2022 Cognite AS */

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/tagger"
)

type tagCmd struct {
	bazelCacheGrpcInsecure bool
	bazelQueryExpression   string
	bazelStderr            bool
	outPath                string
	noPrint                bool
	workspacePath          string
	snapshotName           string
	tagName                string

	storageUrl string

	cmd *cobra.Command
}

func newTagCmd() *tagCmd {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Tag snapshot",
		Long: `Assigns a tag to some (pushed) snapshot, referenced by name. Snapshot name
defaults to the current git HEAD. Tagging a snapshot creates a named
reference to it. For example, a tag "deployed" can be a reference to the
snapshot which was most recently deployed.
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {

		},
	}

	cc := &tagCmd{
		cmd: cmd,
	}

	// bazel flags
	cmd.PersistentFlags().StringVar(&cc.workspacePath, "workspace-path", "", "workspace path")

	// tag flags
	cmd.PersistentFlags().BoolVar(&cc.bazelCacheGrpcInsecure, "bazel-cache-grpc-insecure", true, "use insecure connection for grpc bazel cache")
	cmd.PersistentFlags().StringVar(&cc.bazelQueryExpression, "bazel-query", "//...", "the bazel query expression to consider")
	cmd.PersistentFlags().BoolVar(&cc.bazelStderr, "bazel-stderr", false, "show stderr from bazel")
	cmd.PersistentFlags().StringVar(&cc.outPath, "out-path", "", "output file path")
	cmd.PersistentFlags().BoolVar(&cc.noPrint, "no-print", false, "don't print if not writing to file")
	cmd.PersistentFlags().StringVar(&cc.snapshotName, "name", "", "snapshot name")

	cmd.RunE = cc.runTag

	return cc
}

func (tc *tagCmd) checkArgs(args []string) error {
	// If name is not set, find name from git head
	if tc.snapshotName == "" {
		head, err := getGitHead(tc.workspacePath)
		if err != nil {
			return fmt.Errorf("failed to find name from git: %w", err)
		}
		tc.snapshotName = head
	}

	storageUrl, err := tc.cmd.Flags().GetString("storage-url")
	if err != nil {
		return err
	}
	tc.storageUrl = storageUrl

	tc.tagName = args[0]

	return nil
}

func (tc *tagCmd) runTag(cmd *cobra.Command, args []string) error {
	err := tc.checkArgs(args)
	if err != nil {
		return err
	}

	ctx := context.Background()

	log.Printf("workspace: %s", tc.workspacePath)
	log.Printf("storage:    %s", tc.storageUrl)
	log.Printf("snapshot:  %s", tc.snapshotName)
	log.Printf("tag:       %s", tc.tagName)

	tagArgs := tagger.TagArgs{
		SnapshotName: tc.snapshotName,
		StorageUrl:   tc.storageUrl,
		TagName:      tc.tagName,
	}
	obj, err := tagger.NewTagger().Tag(ctx, &tagArgs)
	if err != nil {
		return err
	}

	log.Printf("tagged snapshot %s as %s: %s", tc.snapshotName, tc.tagName, obj.Path)

	return nil
}
