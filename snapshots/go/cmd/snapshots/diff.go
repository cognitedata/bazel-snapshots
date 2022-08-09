/* Copyright 2022 Cognite AS */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	flag "github.com/spf13/pflag"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/config"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/differ"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/getter"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
)

const (
	outputLabel  = "label"
	outputJSON   = "json"
	outputPretty = "pretty"
)

type diffConfig struct {
	collectConfig
	fromSnapshot *models.Snapshot
	toSnapshot   *models.Snapshot

	outputFormat string
	stderrPretty bool
}

const diffName = "_diff"

func getDiffConfig(c *config.Config) *diffConfig {
	dc := c.Exts[diffName].(*diffConfig)
	dc.collectConfig = *getCollectConfig(c)
	return dc
}

type diffConfigurer struct{}

func (*diffConfigurer) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	dc := &diffConfig{}
	c.Exts[diffName] = dc
	fs.BoolVar(&dc.stderrPretty, "stderr-pretty", false, "pretty-print in stderr in addition")
	fs.StringVar(&dc.outputFormat, "format", outputJSON, "output format (label, json, pretty)")
}

func (*diffConfigurer) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	dc := getDiffConfig(c)

	ctx := context.Background()

	resolveSnapshot := func(name string) (*models.Snapshot, error) {
		// might be a file
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

		return getter.NewGetter().Get(ctx, name, dc.storageURL, false, false)
	}

	if fs.NArg() < 1 || fs.NArg() > 2 {
		return fmt.Errorf("need 1-2 arguments")
	}

	if fromSnapshot, err := resolveSnapshot(fs.Arg(0)); err != nil {
		return fmt.Errorf("failed to get snapshot %s: %w", fs.Arg(0), err)
	} else {
		dc.fromSnapshot = fromSnapshot
	}

	if fs.NArg() == 2 {
		if toSnapshot, err := resolveSnapshot(fs.Arg(1)); err != nil {
			return fmt.Errorf("failed to get snapshot %s: %w", fs.Arg(1), err)
		} else {
			dc.toSnapshot = toSnapshot
		}
	}

	return nil
}

func runDiff(args []string) error {
	cexts := []config.Configurer{
		&bazelConfigurer{},
		&collectConfigurer{},
		&diffConfigurer{},
	}
	c, err := newConfiguration("diff", args, cexts, diffUsage)
	if err != nil {
		return err
	}

	dc := getDiffConfig(c)
	dc.collectConfig = *getCollectConfig(c)

	differ := differ.NewDiffer()
	changes, err := differ.Diff(dc.bazelPath, dc.outPath, dc.queryExpression, dc.workspacePath, dc.bazelCacheGRPCInsecure, dc.bazelStderr, dc.noPrint, dc.fromSnapshot, dc.toSnapshot)
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

func diffUsage(fs *flag.FlagSet) {
	fmt.Fprint(os.Stderr, `usage: diff <from> [<to>]

Compiles a list of the labels which have had changes between two snapshots,
together with what type of change has been made: added, removed or changed.
If only one snapshot is given, then the "to"-snapshot is created from the
current state (see collect). Snapshots can either be files, tags or snapshot
names.

Examples:
	snapshots diff /path/to/snapshot.json
	snapshots diff deployed

FLAGS:
`)
	fs.PrintDefaults()
}
