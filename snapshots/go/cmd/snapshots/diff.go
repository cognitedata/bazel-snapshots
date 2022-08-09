/* Copyright 2022 Cognite AS */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/differ"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/getter"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
)

const (
	outputLabel  = "label"
	outputJSON   = "json"
	outputPretty = "pretty"
)

type diffCmd struct {
	bazelPath              string
	workspacePath          string
	queryExpression        string
	bazelCacheGrpcInsecure bool
	bazelStderr            bool
	outPath                string
	noPrint                bool

	fromSnapshot *models.Snapshot
	toSnapshot   *models.Snapshot

	outputFormat string
	stderrPretty bool

	storageUrl string

	cmd *cobra.Command
}

func newDiffCmd() *diffCmd {
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Diff snapshots",
		Long: `Compiles a list of the labels which have had changes between two snapshots,
together with what type of change has been made: added, removed or changed.
If only one snapshot is given, then the "to"-snapshot is created from the
current state (see collect). Snapshots can either be files, tags or snapshot
names.`,
		Args: cobra.RangeArgs(1, 2),
	}

	dc := &diffCmd{
		cmd: cmd,
	}

	// bazel flags
	cmd.PersistentFlags().StringVar(&dc.bazelPath, "bazel-path", "", "Full URL of the storage")
	cmd.PersistentFlags().StringVar(&dc.workspacePath, "workspace-path", "", "Verbose output")

	// collect flags
	cmd.PersistentFlags().StringVar(&dc.queryExpression, "bazel_query", "//...", "the bazel query expression to consider")
	cmd.PersistentFlags().BoolVar(&dc.bazelCacheGrpcInsecure, "bazel_cache_grpc_insecure", true, "use insecure connection for grpc bazel cache")
	cmd.PersistentFlags().BoolVar(&dc.bazelStderr, "bazel_stderr", false, "show stderr from bazel")
	cmd.PersistentFlags().StringVar(&dc.outPath, "out", "", "output file path")
	cmd.PersistentFlags().BoolVar(&dc.noPrint, "no-print", false, "don't print if not writing to file")

	cmd.RunE = dc.runDiff

	return dc
}

func (*diffCmd) resolveSnapshot(ctx context.Context, name, storageUrl string) (*models.Snapshot, error) {
	// Might be a file
	if _, err := os.Stat(name); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to look for file: %w", err)
	} else if err == nil {
		fileBytes, err := ioutil.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", name, err)
		}
		snapshot := &models.Snapshot{}
		return snapshot, json.Unmarshal(fileBytes, snapshot)
	}

	return getter.NewGetter().Get(ctx, name, storageUrl, false, false)
}

func (dc *diffCmd) checkArgs(args []string) error {
	storageUrl, err := dc.cmd.Flags().GetString("storage-url")
	if err != nil {
		return err
	}
	dc.storageUrl = storageUrl

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

	differ := differ.NewDiffer()
	changes, err := differ.Diff(dc.bazelPath, dc.outPath, dc.queryExpression, dc.workspacePath, dc.bazelCacheGrpcInsecure, dc.bazelStderr, dc.noPrint, dc.fromSnapshot, dc.toSnapshot)
	if err != nil {
		return err
	}

	if dc.stderrPretty {
		if err := differ.DiffOutputPretty(os.Stderr, changes); err != nil {
			return err
		}
	}

	switch dc.outputFormat {
	case outputLabel:
		if err := differ.DiffOutputLabel(os.Stdout, changes); err != nil {
			return err
		}
	case outputJSON:
		if err := differ.DiffOutputJSON(os.Stdout, changes); err != nil {
			return err
		}
	case outputPretty:
		if err := differ.DiffOutputPretty(os.Stdout, changes); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid output format %s", dc.outputFormat)
	}

	return nil
}
