/* Copyright 2022 Cognite AS */

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"

	flag "github.com/spf13/pflag"
	"google.golang.org/grpc"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/bazel"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/config"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
)

type collectConfig struct {
	bazelConfig
	commonConfig
	queryExpression        string
	bazelCacheGRPCInsecure bool
	bazelStderr            bool
	outPath                string
	push                   bool
	noPrint                bool
}

const collectName = "_collect"

func getCollectConfig(c *config.Config) *collectConfig {
	cc := c.Exts[collectName].(*collectConfig)
	cc.bazelConfig = *getBazelConfig(c)
	cc.commonConfig = *getCommonConfig(c)
	return cc
}

type collectConfigurer struct{}

func (*collectConfigurer) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	cc := &collectConfig{}
	c.Exts[collectName] = cc
	fs.StringVar(&cc.queryExpression, "bazel_query", "//...", "the bazel query expression to consider")
	fs.BoolVar(&cc.bazelCacheGRPCInsecure, "bazel_cache_grpc_insecure", true, "use insecure connection for grpc bazel cache")
	fs.BoolVar(&cc.bazelStderr, "bazel_stderr", false, "show stderr from bazel")
	fs.StringVar(&cc.outPath, "out", "", "output file path")
	fs.BoolVar(&cc.noPrint, "no-print", false, "don't print if not writing to file")
}

func (*collectConfigurer) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	cc := getCollectConfig(c)

	// if it's a relative path, assume workspace-relative. The command is
	// probably run with `bazel run`, and we don't know from where.
	if cc.outPath != "" && !path.IsAbs(cc.outPath) {
		cc.outPath = path.Join(cc.workspacePath, cc.outPath)
	}

	return nil
}

func runCollect(args []string) error {
	cexts := []config.Configurer{
		&bazelConfigurer{},
		&collectConfigurer{},
	}
	c, err := newConfiguration("collect", args, cexts, collectUsage)
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	cc := getCollectConfig(c)

	log.Println("bazel path:      ", cc.bazelPath)
	log.Println("workspace path:  ", cc.workspacePath)
	log.Println("query expression:", cc.queryExpression)
	log.Println("out path:        ", cc.outPath)

	// run the command
	ctx := context.Background()
	manifest, err := collect(cc)
	if err != nil {
		return fmt.Errorf("failed to collect: %w", err)
	}

	if cc.push {
		pc := getPushConfig(c)
		pc.snapshot = manifest

		obj, err := push(ctx, pc)
		if err != nil {
			return fmt.Errorf("failed to push snapshot: %w", err)
		}

		log.Printf("pushed snapshot of %d bytes: %s", obj.MustGetContentLength(), obj.Path)
	}

	return nil
}

// collect uses Bazel directly to build and collect all change tracker files to
// compose a snapshot. It first runs 'bazel build' with a query expression, e.g.
// '//...', with the change_track_files output groups, while also capturing
// build events (see Bazel's --build_event_json_file). It then retrieves all
// these tracker files, parses them and builds the snapshot.
func collect(cc *collectConfig) (*models.Snapshot, error) {
	dialOptions := []grpc.DialOption{}
	if cc.bazelCacheGRPCInsecure {
		dialOptions = append(dialOptions, grpc.WithInsecure())
	}

	bstderr := io.Discard
	if cc.bazelStderr {
		bstderr = os.Stderr
	}

	ctx := context.Background()
	bazelc := bazel.NewClient(cc.bazelPath, cc.workspacePath, bstderr)
	bcache := bazel.NewDefaultDelegatingCache(dialOptions)

	// build digests, get the build events
	log.Println("collecting digests")
	buildEvents, err := bazelc.BuildEventOutput(ctx, cc.queryExpression, "--output_groups=change_track_files")
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

	if cc.outPath != "" {
		// write to outpath
		outFile, err := os.Create(cc.outPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open out path: %w", err)
		}

		if _, err := io.Copy(outFile, bytes.NewReader(snapshotJSON)); err != nil {
			return nil, err
		}
		log.Printf("wrote file to %s", outFile.Name())
	}
	if cc.outPath == "" && !cc.noPrint {
		// write to stdout
		if _, err := io.Copy(os.Stdout, bytes.NewBuffer(snapshotJSON)); err != nil {
			return nil, err
		}
	}

	return manifest, nil
}

func collectUsage(fs *flag.FlagSet) {
	fmt.Fprint(os.Stderr, `usage: collect

Creates a snapshot from the current state and writes it to stdout or to a
file. Collects all digests by building //... with the 'change_track_files'
output group. Observes the build events to find the relevant files. Compiles
all the digest files to a snapshot.

FLAGS:
`)
	fs.PrintDefaults()
}
