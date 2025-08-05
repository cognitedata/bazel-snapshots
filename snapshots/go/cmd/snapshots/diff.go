/* Copyright 2022 Cognite AS */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/spf13/cobra"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/differ"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/getter"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
)

type diffCmd struct {
	bazelCacheGrpcInsecure bool
	bazelCacheGrpcMetadata []string
	bazelPath              string
	bazelQueryExpression   string
	bazelRcPath            string
	bazelStderr            bool
	buildEventsPath        string
	credentialHelper       string
	outPath                string
	noPrint                bool
	workspacePath          string

	fromSnapshot *models.Snapshot
	toSnapshot   *models.Snapshot

	outputFormat OutputFormat
	stderrPretty bool

	storageUrl string

	cmd *cobra.Command
}

func newDiffCmd() *diffCmd {
	cmd := &cobra.Command{
		Use:   "diff --format=FORMAT FROM [TO]",
		Short: "Diff snapshots",
		Long: `Compiles a list of the labels which have had changes between two snapshots (FROM and TO),
together with what type of change has been made: added, removed or changed.
If only the FROM snapshot is given, then the TO snapshot is created from the
current state (see collect). Snapshots can either be files, tags or snapshot
names.`,
		Args: cobra.RangeArgs(1, 2),
	}

	dc := &diffCmd{
		cmd: cmd,
	}

	// bazel flags
	cmd.PersistentFlags().StringVar(&dc.bazelPath, "bazel-path", "", "path to the bazel executable")
	cmd.PersistentFlags().StringVar(&dc.bazelRcPath, "bazelrc", "", ".bazelrc path")
	cmd.PersistentFlags().StringVar(&dc.workspacePath, "workspace-path", "", "workspace path")

	// collect flags
	cmd.PersistentFlags().BoolVar(&dc.bazelCacheGrpcInsecure, "bazel_cache_grpc_insecure", false, "use insecure connection for grpc bazel cache")
	cmd.PersistentFlags().StringArrayVar(&dc.bazelCacheGrpcMetadata, "bazel_cache_grpc_metadata", []string{}, "add metadata to connection for grpc bazel cache")
	cmd.PersistentFlags().StringVar(&dc.bazelQueryExpression, "bazel-query", "//...", "the bazel query expression to consider")
	cmd.PersistentFlags().StringVar(&dc.buildEventsPath, "build_event_json_file", "", "a bazel build event json file")
	cmd.PersistentFlags().BoolVar(&dc.bazelStderr, "bazel_stderr", false, "show stderr from bazel")
	cmd.PersistentFlags().StringVar(&dc.credentialHelper, "credential_helper", "", "path to a credential helper, relative to workspace-path")
	cmd.PersistentFlags().Var(&dc.outputFormat, "format", "output format")
	cmd.PersistentFlags().StringVar(&dc.outPath, "out", "", "output file path")
	cmd.PersistentFlags().BoolVar(&dc.noPrint, "no-print", false, "don't print if not writing to file")
	cmd.PersistentFlags().BoolVar(&dc.stderrPretty, "stderr-pretty", false, "pretty-print in stderr in addition")

	cmd.RunE = dc.runDiff

	return dc
}

func (dc *diffCmd) resolveSnapshot(ctx context.Context, name, storageUrl string) (*models.Snapshot, error) {
	// Might be a file
	if _, err := os.Stat(name); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to look for file: %w", err)
	} else if err == nil {
		fileBytes, err := os.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", name, err)
		}
		snapshot := &models.Snapshot{}
		return snapshot, json.Unmarshal(fileBytes, snapshot)
	}

	getArgs := getter.GetArgs{
		Name:       name,
		StorageUrl: dc.storageUrl,
		SkipNames:  false,
		SkipTags:   false,
	}
	return getter.NewGetter().Get(ctx, &getArgs)
}

func (dc *diffCmd) checkArgs(args []string) error {
	if dc.bazelPath == "" {
		path, err := exec.LookPath("bazel")
		if err != nil {
			return err
		}
		dc.bazelPath = path
	}

	storageUrl, err := dc.cmd.Flags().GetString("storage-url")
	if err != nil {
		return err
	}
	dc.storageUrl = storageUrl

	if dc.workspacePath == "" {
		if wsDir := os.Getenv("BUILD_WORKSPACE_DIRECTORY"); wsDir != "" {
			dc.workspacePath = wsDir
		} else {
			return fmt.Errorf("--workspace-path not specified and BUILD_WORKSPACE_DIRECTORY not set")
		}
	}

	if dc.outPath != "" && !path.IsAbs(dc.outPath) {
		dc.outPath = path.Join(dc.workspacePath, dc.outPath)
	}

	if dc.bazelRcPath != "" && !path.IsAbs(dc.bazelRcPath) {
		dc.bazelRcPath = path.Join(dc.workspacePath, dc.bazelRcPath)
	}

	return nil
}

func (dc *diffCmd) runDiff(cmd *cobra.Command, args []string) error {
	err := dc.checkArgs(args)
	if err != nil {
		return err
	}

	ctx := context.Background()

	fromSnapshotName := args[0]
	if fromSnapshot, err := dc.resolveSnapshot(ctx, fromSnapshotName, dc.storageUrl); err != nil {
		return fmt.Errorf("failed to get snapshot %s: %w", fromSnapshotName, err)
	} else {
		dc.fromSnapshot = fromSnapshot
	}

	if len(args) == 2 {
		toSnapshotName := args[1]
		if toSnapshot, err := dc.resolveSnapshot(ctx, toSnapshotName, dc.storageUrl); err != nil {
			return fmt.Errorf("failed to get snapshot %s: %w", toSnapshotName, err)
		} else {
			dc.toSnapshot = toSnapshot
		}
	}

	diff := differ.NewDiffer()
	diffArgs := differ.DiffArgs{
		BazelCacheGrpcs:        !dc.bazelCacheGrpcInsecure,
		BazelCacheGrpcMetadata: dc.bazelCacheGrpcMetadata,
		BazelExpression:        dc.bazelQueryExpression,
		BazelPath:              dc.bazelPath,
		BazelRcPath:            dc.bazelRcPath,
		BazelWorkspacePath:     dc.workspacePath,
		BazelWriteStderr:       dc.bazelStderr,
		BuildEventsPath:        dc.buildEventsPath,
		CredentialHelper:       dc.credentialHelper,
		OutPath:                dc.outPath,
		NoPrint:                dc.noPrint,
		FromSnapshot:           dc.fromSnapshot,
		ToSnapshot:             dc.toSnapshot,
	}

	changes, err := diff.Diff(&diffArgs)
	if err != nil {
		return err
	}

	if dc.stderrPretty {
		if err := diff.DiffOutputPretty(os.Stderr, changes); err != nil {
			return err
		}
	}

	switch dc.outputFormat {
	case formatLabel:
		if err := diff.DiffOutputLabel(os.Stdout, changes); err != nil {
			return err
		}
	case formatJson:
		if err := diff.DiffOutputJSON(os.Stdout, changes); err != nil {
			return err
		}
	case formatPretty:
		if err := diff.DiffOutputPretty(os.Stdout, changes); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid output format %s", dc.outputFormat)
	}

	return nil
}
