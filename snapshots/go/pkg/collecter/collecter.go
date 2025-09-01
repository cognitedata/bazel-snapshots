package collecter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"log"
	"os"
	"strings"

	"google.golang.org/grpc/metadata"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/bazel"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/cache"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
)

type collecter struct{}

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
	CredentialHelper       string
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

	ctx := context.Background()

	var credential string
	if args.CredentialHelper != "" {
		creds, err := getAuthorization(args.CredentialHelper, args.BazelWorkspacePath)
		if err != nil {
			return nil, fmt.Errorf("error getting credentials: %w", err)
		}
		credential = creds[0]
	}

	bcache := cache.NewDefaultDelegatingCache(credential)

	// build digests, get the build events
	log.Printf("collecting digests from %s", args.BazelExpression)
	bazelArgs := []string{args.BazelExpression, "--output_groups=change_track_files"}

	var buildEvents iter.Seq2[bazel.BuildEventOutput, error]
	if args.BazelBuildEventsPath != "" {
		f, err := os.Open(args.BazelBuildEventsPath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = f.Close() }()

		buildEvents = bazel.ParseBuildEventsFile(f)
	} else {
		bazelc := bazel.NewClient(args.BazelPath, args.BazelWorkspacePath, bstderr)
		buildEvents = bazelc.BuildEventOutput(ctx, args.BazelRcPath, bazelArgs...)
	}

	bazelFiles := make(namedSetsOfFiles)
	labelFiles := make(map[string]string) // label -> uri
	for event, err := range buildEvents {
		if err != nil {
			return nil, fmt.Errorf("error reading build event: %w", err)
		}
		switch {
		case event.ID.NamedSet.ID != "":
			bazelFiles.Put(event.ID.NamedSet.ID, event.NamedSetOfFiles)

		case event.ID.TargetCompleted.Label != "":
			label := event.ID.TargetCompleted.Label
			var labelURI string
		uriSearch:
			for _, group := range event.Completed.OutputGroups {
				if group.Name != "change_track_files" {
					continue
				}

				// Found a change_track_files output group.
				// Find the file set for this label.
				// File sets can point to other file sets,
				// so we need to traverse them
				// until we find the actual change tracker files.
				//
				// Bazel guarantees that a file set will be
				// provided before it's referenced.
				for _, fileSet := range group.FileSets {
					for uri := range bazelFiles.ByID(fileSet.ID) {
						// change_track_files will
						// contain only one file per label,
						// so we can stop at the first one.
						labelURI = uri
						break uriSearch
					}
				}
			}
			if label != "" && labelURI != "" {
				labelFiles[label] = labelURI
			}
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

// namedSetsOfFiles is an in-memory buffer for NamedSetOfFiles.
//
// Bazel produces NamedSetOfFiles events in the build event stream,
// alongside TargetCompleted events.
// Both events can reference other NamedSetOfFiles by ID.
// Bazel guarantees that a NamedSetOfFiles event will be
// provided before it's referenced.
//
// To use namedSetsOfFiles, populate it with NamedSetOfFiles events
// as they are produced by Bazel using Put().
//
// To retrieve all files in a NamedSetOfFiles by ID,
// use ByID(), to get an iterator over the file URIs.
type namedSetsOfFiles map[string]bazel.NamedSetOfFiles

func (fs namedSetsOfFiles) Put(id string, ns bazel.NamedSetOfFiles) {
	fs[id] = ns
}

func (fs namedSetsOfFiles) ByID(fileSetID string) iter.Seq[string] {
	return func(yield func(string) bool) {
		pending := []string{fileSetID}
		seen := make(map[string]struct{})
		for len(pending) > 0 {
			currentID := pending[len(pending)-1]
			pending = pending[:len(pending)-1]

			if _, ok := seen[currentID]; ok {
				continue // already seen this file set
			}
			seen[currentID] = struct{}{}

			for _, file := range fs[currentID].Files {
				if !yield(file.URI) {
					return
				}
			}

			for _, fileSet := range fs[currentID].FileSets {
				pending = append(pending, fileSet.ID)
			}
		}
	}
}

func createMetadata(input []string) metadata.MD {
	md := metadata.MD{}

	for _, data := range input {
		s := strings.SplitN(data, "=", 2)
		md.Append(s[0], s[1])
	}

	return md
}
