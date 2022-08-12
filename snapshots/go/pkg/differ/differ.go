package differ

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/collecter"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
)

type differ struct{}

func NewDiffer() *differ {
	return &differ{}
}

type DiffArgs struct {
	BazelCacheGrpcs        bool
	BazelCacheGrpcMetadata []string
	BazelExpression        string
	BazelPath              string
	BazelRcPath            string
	BazelWorkspacePath     string
	BazelWriteStderr       bool
	OutPath                string
	NoPrint                bool
	FromSnapshot           *models.Snapshot
	ToSnapshot             *models.Snapshot
}

func (*differ) Diff(args *DiffArgs) ([]models.TrackerChange, error) {
	// if toSnapshot is not set, then run collect
	if args.ToSnapshot == nil {
		collectArgs := collecter.CollectArgs{
			BazelCacheGrpcs:        args.BazelCacheGrpcs,
			BazelCacheGrpcMetadata: args.BazelCacheGrpcMetadata,
			BazelExpression:        args.BazelExpression,
			BazelPath:              args.BazelPath,
			BazelRcPath:            args.BazelRcPath,
			BazelWorkspacePath:     args.BazelWorkspacePath,
			BazelWriteStderr:       args.BazelWriteStderr,
			OutPath:                args.OutPath,
			NoPrint:                args.NoPrint,
		}
		snapshot, err := collecter.NewCollecter().Collect(&collectArgs)
		if err != nil {
			return nil, err
		}

		args.ToSnapshot = snapshot
	}

	// create a map with all labels
	allLabels := make(map[string]bool)
	for label := range args.FromSnapshot.Labels {
		allLabels[label] = true
	}
	for label := range args.ToSnapshot.Labels {
		allLabels[label] = true
	}

	changes := make([]models.TrackerChange, 0, len(allLabels))

	for label := range allLabels {
		fromTracker := args.FromSnapshot.Labels[label]
		toTracker := args.ToSnapshot.Labels[label]

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

// diffOutputLabel writes added or changed labels, one per line.
func (*differ) DiffOutputLabel(dest io.Writer, changes []models.TrackerChange) error {
	for _, change := range changes {
		if change.ChangeType == models.Added || change.ChangeType == models.Changed {
			fmt.Fprintf(dest, "%s\n", change.Label)
		}
	}
	return nil
}

// diffOutputJSON writes added, changed or removed TrackerChanges as a JSON list.
func (*differ) DiffOutputJSON(dest io.Writer, changes []models.TrackerChange) error {
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
func (*differ) DiffOutputPretty(dest io.Writer, changes []models.TrackerChange) error {
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
