/* Copyright 2022 Cognite AS */

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/collecter"
)

type collectCmd struct {
	bazelCacheGrpcInsecure bool
	bazelCacheGrpcMetadata []string
	bazelPath              string
	bazelQueryExpression   string
	bazelRcPath            string
	bazelStderr            bool
	bazelStdout            bool
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
	cmd.PersistentFlags().StringVar(&cc.bazelPath, "bazel-path", "", "path to the bazel executable")
	cmd.PersistentFlags().StringVar(&cc.bazelRcPath, "bazelrc", "", ".bazelrc path")
	cmd.PersistentFlags().StringVar(&cc.workspacePath, "workspace-path", "", "workspace path")

	// collect flags
	cmd.PersistentFlags().BoolVar(&cc.bazelCacheGrpcInsecure, "bazel_cache_grpc_insecure", false, "use insecure connection for grpc bazel cache")
	cmd.PersistentFlags().StringArrayVar(&cc.bazelCacheGrpcMetadata, "bazel_cache_grpc_metadata", []string{}, "add metadata to connection for grpc bazel cache")
	cmd.PersistentFlags().StringVar(&cc.bazelQueryExpression, "bazel-query", "//...", "the bazel query expression to consider")
	cmd.PersistentFlags().BoolVar(&cc.bazelStderr, "bazel-stderr", false, "show stderr from bazel")
	cmd.PersistentFlags().BoolVar(&cc.bazelStdout, "bazel-stdout", false, "show stdout from bazel")
	cmd.PersistentFlags().StringVar(&cc.outPath, "out-path", "", "output file path")
	cmd.PersistentFlags().BoolVar(&cc.noPrint, "no-print", false, "don't print if not writing to file")

	cmd.RunE = cc.runCollect

	return cc
}

func (cc *collectCmd) checkArgs(args []string) error {
	if cc.bazelPath == "" {
		path, err := exec.LookPath("bazel")
		if err != nil {
			return err
		}
		cc.bazelPath = path
	}

	if cc.workspacePath == "" {
		if wsDir := os.Getenv("BUILD_WORKSPACE_DIRECTORY"); wsDir != "" {
			cc.workspacePath = wsDir
		} else {
			return fmt.Errorf("--workspace-path not specified and BUILD_WORKSPACE_DIRECTORY not set")
		}
	}

	if cc.outPath != "" && !path.IsAbs(cc.outPath) {
		cc.outPath = path.Join(cc.workspacePath, cc.outPath)
	}

	if cc.bazelRcPath != "" && !path.IsAbs(cc.bazelRcPath) {
		cc.bazelRcPath = path.Join(cc.workspacePath, cc.bazelRcPath)
	}

	for _, md := range cc.bazelCacheGrpcMetadata {
		s := strings.SplitN(md, "=", 2)
		if len(s) != 2 {
			return fmt.Errorf("--bazel_cache_grpc_metadata must be in format key=value: %s", md)
		}
	}

	return nil
}

func (cc *collectCmd) runCollect(cmd *cobra.Command, args []string) error {
	err := cc.checkArgs(args)
	if err != nil {
		return err
	}

	log.Println("bazel path:      ", cc.bazelPath)
	log.Println("bazelrc path:    ", cc.bazelRcPath)
	log.Println("workspace path:  ", cc.workspacePath)
	log.Println("query expression:", cc.bazelQueryExpression)
	log.Println("out path:        ", cc.outPath)

	collectArgs := collecter.CollectArgs{
		BazelCacheGrpcs:        !cc.bazelCacheGrpcInsecure,
		BazelCacheGrpcMetadata: cc.bazelCacheGrpcMetadata,
		BazelExpression:        cc.bazelQueryExpression,
		BazelPath:              cc.bazelPath,
		BazelRcPath:            cc.bazelRcPath,
		BazelWorkspacePath:     cc.workspacePath,
		BazelWriteStderr:       cc.bazelStderr,
		BazelWriteStdout:       cc.bazelStdout,
		OutPath:                cc.outPath,
		NoPrint:                cc.noPrint,
	}
	if _, err := collecter.NewCollecter().Collect(&collectArgs); err != nil {
		return fmt.Errorf("failed to collect: %w", err)
	}

	return nil
}
