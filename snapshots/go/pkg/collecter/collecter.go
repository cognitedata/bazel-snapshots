package collecter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"google.golang.org/grpc"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/bazel"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
)

type collecter struct {
}

func NewCollecter() *collecter {
	return &collecter{}
}

// collect uses Bazel directly to build and collect all change tracker files to
// compose a snapshot. It first runs 'bazel build' with a query expression, e.g.
// '//...', with the change_track_files output groups, while also capturing
// build events (see Bazel's --build_event_json_file). It then retrieves all
// these tracker files, parses them and builds the snapshot.
func (c *collecter) Collect(bazelPath, outPath, queryExpression, workspacePath string, bazelCacheGrpcInsecure, bazelStderr, noPrint bool) (*models.Snapshot, error) {
	if bazelPath == "" {
		path, err := exec.LookPath("bazel")
		if err != nil {
			return nil, err
		}

		bazelPath = path
	}

	if workspacePath == "" {
		if wsDir := os.Getenv("BUILD_WORKSPACE_DIRECTORY"); wsDir != "" {
			workspacePath = wsDir
		} else {
			return nil, fmt.Errorf("workspace-path not specified and BUILD_WORKSPACE_DIRECTORY not set")
		}
	}

	dialOptions := []grpc.DialOption{}
	if bazelCacheGrpcInsecure {
		dialOptions = append(dialOptions, grpc.WithInsecure())
	}

	bstderr := io.Discard
	if bazelStderr {
		bstderr = os.Stderr
	}

	ctx := context.Background()
	bazelc := bazel.NewClient(bazelPath, workspacePath, bstderr)
	bcache := bazel.NewDefaultDelegatingCache(dialOptions)

	// build digests, get the build events
	log.Printf("collecting digests from %s", queryExpression)
	buildEvents, err := bazelc.BuildEventOutput(ctx, queryExpression, "--output_groups=change_track_files")
	if err != nil {
		return nil, err
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
		// retrieve the content from cache
		trackerContent, err := bcache.Read(ctx, uri)
		if err != nil {
			return nil, fmt.Errorf("failed to get item %s from cache: %w", uri, err)
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

	if outPath != "" {
		// write to outpath
		outFile, err := os.Create(outPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open out path: %w", err)
		}

		if _, err := io.Copy(outFile, bytes.NewReader(snapshotJSON)); err != nil {
			return nil, err
		}
		log.Printf("wrote file to %s", outFile.Name())
	}
	if outPath == "" && !noPrint {
		// write to stdout
		if _, err := io.Copy(os.Stdout, bytes.NewBuffer(snapshotJSON)); err != nil {
			return nil, err
		}
	}

	return manifest, nil
}
