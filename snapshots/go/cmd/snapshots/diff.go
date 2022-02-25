/* Copyright 2022 Cognite AS */

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
	flag "github.com/spf13/pflag"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/config"
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
	fs.StringVar(&dc.outputFormat, "format", outputLabel, "output format (label, json, pretty)")
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

		return get(ctx, &getConfig{commonConfig: dc.commonConfig, name: name})
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

	changes, err := diff(dc)
	if err != nil {
		return err
	}

	if dc.stderrPretty {
		if err := diffOutputPretty(os.Stderr, changes); err != nil {
			return err
		}
	}

	switch dc.outputFormat {
	case outputLabel:
		if err := diffOutputLabel(os.Stdout, changes); err != nil {
			return err
		}
	case outputJSON:
		if err := diffOutputJSON(os.Stdout, changes); err != nil {
			return err
		}
	case outputPretty:
		if err := diffOutputPretty(os.Stdout, changes); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid output format %s", dc.outputFormat)
	}

	return nil
}

// diffOutputLabel writes added or changed labels, one per line.
func diffOutputLabel(dest io.Writer, changes []models.TrackerChange) error {
	for _, change := range changes {
		if change.ChangeType == models.Added || change.ChangeType == models.Changed {
			fmt.Fprintf(dest, "%s\n", change.Label)
		}
	}
	return nil
}

// diffOutputJSON writes added, changed or removed TrackerChanges as a JSON list.
func diffOutputJSON(dest io.Writer, changes []models.TrackerChange) error {
	changedOrAdded := make([]models.TrackerChange, 0, len(changes))
	for _, change := range changes {
		if change.ChangeType != models.Unchanged {
			changedOrAdded = append(changedOrAdded, change)
		}
	}

	out, err := json.MarshalIndent(changedOrAdded, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal changes: %w", err)
	}

	_, err = io.Copy(dest, bytes.NewReader(out))
	return err
}

// diffOutputPretty writes a human-readable table of added, changed or removed trackers.
func diffOutputPretty(dest io.Writer, changes []models.TrackerChange) error {
	table := tablewriter.NewWriter(dest)
	table.SetAutoMergeCells(true)
	table.SetRowLine(true)

	// a somewhat arbitrary sorting algorithm attempting to make things pretty
	for _, change := range changes {
		sort.Strings(change.Tags)
	}
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].ChangeType.String() != changes[j].ChangeType.String() {
			return changes[i].ChangeType.String() < changes[j].ChangeType.String()
		}
		if len(changes[i].Tags) != len(changes[j].Tags) {
			return len(changes[i].Tags) < len(changes[j].Tags)
		}
		for idx := range changes[i].Tags {
			if len(changes[j].Tags) >= idx+1 {
				if changes[i].Tags[idx] != changes[j].Tags[idx] {
					return changes[i].Tags[idx] < changes[j].Tags[idx]
				}
			}
		}
		return changes[i].Label < changes[j].Label
	})

	table.SetHeader([]string{"Change", "Tags", "Label"})
	for _, change := range changes {
		if change.ChangeType == models.Added || change.ChangeType == models.Changed || change.ChangeType == models.Removed {
			table.Append([]string{
				change.ChangeType.String(),
				strings.Join(change.Tags, "\n"),
				change.Label,
			})
		}
	}

	table.Render()
	return nil
}

func diff(dc *diffConfig) ([]models.TrackerChange, error) {
	// if toSnapshot is not set, then run collect
	if dc.toSnapshot == nil {
		dc.collectConfig.noPrint = true
		toSnapshot, err := collect(&dc.collectConfig)
		if err != nil {
			return nil, err
		} else {
			dc.toSnapshot = toSnapshot
		}
	}

	// create a map with all labels
	allLabels := make(map[string]bool)
	for label := range dc.fromSnapshot.Labels {
		allLabels[label] = true
	}
	for label := range dc.toSnapshot.Labels {
		allLabels[label] = true
	}

	changes := make([]models.TrackerChange, 0, len(allLabels))

	for label := range allLabels {
		fromTracker := dc.fromSnapshot.Labels[label]
		toTracker := dc.toSnapshot.Labels[label]

		change := models.TrackerChange{
			Label: label,
		}

		if toTracker != nil {
			change.Tracker = *toTracker
		}

		if fromTracker == nil {
			change.ChangeType = models.Added
		} else if toTracker == nil {
			change.ChangeType = models.Removed
		} else if fromTracker.Digest != toTracker.Digest {
			change.ChangeType = models.Changed
		} else {
			change.ChangeType = models.Unchanged
		}

		changes = append(changes, change)
	}

	return changes, nil
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
