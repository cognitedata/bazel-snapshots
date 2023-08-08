package collecter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"google.golang.org/grpc/metadata"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/bazel"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/cache"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
)

type collecter struct {
}

func NewCollecter() *collecter {
	return &collecter{}
}

type CollectArgs struct {
	BazelCacheGrpcs        bool
	BazelCacheGrpcMetadata []string
	BazelExpression        string
	BazelPath              string
	BazelRcPath            string
	BazelWorkspacePath     string
	BazelWriteStderr       bool
	BazelBuildEventsPath   string
	OutPath                string
	NoPrint                bool
}

// collect uses Bazel directly to build and collect all change tracker files to
// compose a snapshot. It first runs 'bazel build' with a query expression, e.g.
// '//...', with the change_track_files output groups, while also capturing
// build events (see Bazel's --build_event_json_file). It then retrieves all
// these tracker files, parses them and builds the snapshot.
func (c *collecter) Collect(args *CollectArgs) (*models.Snapshot, error) {
	bstderr := io.Discard
	if args.BazelWriteStderr {
		bstderr = os.Stderr
	}

	var buildEvents []bazel.BuildEventOutput
	ctx := context.Background()
	bcache := cache.NewDefaultDelegatingCache()

	if args.BazelBuildEventsPath != "" {
  // get digests from file
	 log.Printf("collecting digests from %s", args.BazelBuildEventsPath)

		f, err := os.Open(args.BazelBuildEventsPath)
		if err != nil {
			return nil, err
		}
		events, err := bazel.ParseBuildEventsFile(f)
		if err != nil {
			return nil, err
		}
		buildEvents = events
	} else {
		bazelc := bazel.NewClient(args.BazelPath, args.BazelWorkspacePath, bstderr)

	 // build digests, get the build events
	 log.Printf("collecting digests from %s", args.BazelExpression)
	 bazelArgs := []string{args.BazelExpression, "--output_groups=change_track_files"}

		events, err := bazelc.BuildEventOutput(ctx, args.BazelRcPath, bazelArgs...)
		if err != nil {
			return nil, err
		}
		buildEvents = events
	}

	log.Printf("got %d build events", len(buildEvents))

	// create a map from label to file
	labelFiles := map[string]string{}
	for _, event := range buildEvents {
		label := event.ID.TargetCompleted.Label
		var uri string

		for idx, g := range event.Completed.OutputGroups {
			if g.Name == "change_track_files" {
				uri = event.Completed.ImportantOutput[idx].URI
				break
			}
		}

		if label != "" && uri != "" {
			labelFiles[label] = uri
		}
	}
	log.Printf("got %d change trackers", len(labelFiles))

	manifest := &models.Snapshot{
		Labels: map[string]*models.Tracker{},
	}

	// populate manifest labels
	for label, uri := range labelFiles {
		// add cache metadata (headers) to request
		ctx = metadata.NewOutgoingContext(ctx, createMetadata(args.BazelCacheGrpcMetadata))

		// retrieve the content from cache
		trackerContent, err := bcache.Read(ctx, args.BazelCacheGrpcs, uri)
		if err != nil {
			return nil, fmt.Errorf("failed to get item %s for label %s from cache: %w", uri, label, err)
		}

		tracker := &models.Tracker{}
		if err := json.Unmarshal(trackerContent, &tracker); err != nil {
			return nil, fmt.Errorf("invalid tracker content %s: %w", trackerContent, err)
		}

		manifest.Labels[label] = tracker
	}

	// should support writing to outfile here, since it can be reused in other commands
	snapshotJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest JSON: %w", err)
	}

	if args.OutPath != "" {
		// write to outpath
		outFile, err := os.Create(args.OutPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open out path: %w", err)
		}

		if _, err := io.Copy(outFile, bytes.NewReader(snapshotJSON)); err != nil {
			return nil, err
		}
		log.Printf("wrote file to %s", outFile.Name())
	}

	if args.OutPath == "" && !args.NoPrint {
		// write to stdout
		if _, err := io.Copy(os.Stdout, bytes.NewBuffer(snapshotJSON)); err != nil {
			return nil, err
		}
	}

	return manifest, nil
}

func createMetadata(input []string) metadata.MD {
	md := metadata.MD{}

	for _, data := range input {
		s := strings.SplitN(data, "=", 2)
		md.Append(s[0], s[1])
	}

	return md
}
