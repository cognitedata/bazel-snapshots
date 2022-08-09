/* Copyright 2022 Cognite AS */

package main

import (
	"fmt"
	"path"

	"github.com/spf13/cobra"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/collecter"
)

type collectCmd struct {
	bazelCacheGrpcInsecure bool
	bazelPath              string
	bazelQueryExpression   string
	bazelStderr            bool
	outPath                string
	noPrint                bool
	workspacePath          string

	cmd *cobra.Command
}

func newCollectCmd() *collectCmd {
	cmd := &cobra.Command{
		Use:   "collect",
		Short: "Collect digests",
		Long: `Creates a snapshot from the current state and writes it to stdout or to a
	file. Collects all digests by building //... with the 'change_track_files'
	output group. Observes the build events to find the relevant files. Compiles
	all the digest files to a snapshot.`,
		Run: func(cmd *cobra.Command, args []string) {

		},
	}

	cc := &collectCmd{
		cmd: cmd,
	}

	// bazel flags
	cmd.PersistentFlags().StringVar(&cc.bazelPath, "bazel-path", "", "Full URL of the storage")
	cmd.PersistentFlags().StringVar(&cc.workspacePath, "workspace-path", "", "Verbose output")

	// collect flags
	cmd.PersistentFlags().BoolVar(&cc.bazelCacheGrpcInsecure, "bazel-cache-grpc-insecure", true, "use insecure connection for grpc bazel cache")
	cmd.PersistentFlags().StringVar(&cc.bazelQueryExpression, "bazel-query", "//...", "the bazel query expression to consider")
	cmd.PersistentFlags().BoolVar(&cc.bazelStderr, "bazel-stderr", false, "show stderr from bazel")
	cmd.PersistentFlags().StringVar(&cc.outPath, "out-path", "", "output file path")
	cmd.PersistentFlags().BoolVar(&cc.noPrint, "no-print", false, "don't print if not writing to file")

	cmd.RunE = cc.runCollect

	return cc
}

func (cc *collectCmd) checkArgs(args []string) error {
	if cc.outPath != "" && !path.IsAbs(cc.outPath) {
		cc.outPath = path.Join(cc.workspacePath, cc.outPath)
	}

	return nil
}

func (cc *collectCmd) runCollect(cmd *cobra.Command, args []string) error {
	err := cc.checkArgs(args)
	if err != nil {
		return err
	}

	if _, err := collecter.NewCollecter().Collect(cc.bazelPath, cc.outPath, cc.bazelQueryExpression, cc.workspacePath, cc.bazelCacheGrpcInsecure, cc.bazelStderr, cc.noPrint); err != nil {
		return fmt.Errorf("failed to collect: %w", err)
	}

	return nil
}
