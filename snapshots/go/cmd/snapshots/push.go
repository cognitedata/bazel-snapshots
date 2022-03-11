/* Copyright 2022 Cognite AS */

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	flag "github.com/spf13/pflag"
	"go.beyondstorage.io/v5/types"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/config"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/storage"
)

type pushConfig struct {
	commonConfig
	bazelConfig  // needed for workspace path
	snapshotPath string
	snapshot     *models.Snapshot
	name         string
}

const pushName = "_push"

func getPushConfig(c *config.Config) *pushConfig {
	pc := c.Exts[pushName].(*pushConfig)
	pc.bazelConfig = *getBazelConfig(c)
	pc.commonConfig = *getCommonConfig(c)
	return pc
}

type pushConfigurer struct{}

func (*pushConfigurer) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	pc := &pushConfig{}
	c.Exts[pushName] = pc
	fs.StringVar(&pc.snapshotPath, "snapshot-path", "", "path to snapshot to be pushed")
	fs.StringVar(&pc.name, "name", "", "snapshot name (defaults to HEAD git sha)")
}

func (*pushConfigurer) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	pc := getPushConfig(c)

	// if name is not set, find name from git head
	if pc.name == "" {
		head, err := getGitHead(pc.workspacePath)
		if err != nil {
			return fmt.Errorf("failed to find name from git: %w", err)
		}
		pc.name = head
	}

	// read the manifest
	if pc.snapshot == nil && pc.snapshotPath != "" {
		// if it's a relative path, assume workspace-relative. The command is
		// probably run with `bazel run`, and we don't know from where.
		if !path.IsAbs(pc.snapshotPath) {
			pc.snapshotPath = path.Join(pc.workspacePath, pc.snapshotPath)
		}

		log.Println("reading snapshot from", pc.snapshotPath)
		pc.snapshot = &models.Snapshot{}
		contents, err := ioutil.ReadFile(pc.snapshotPath)
		if err != nil {
			return fmt.Errorf("failed to read snapshot path: %w", err)
		}
		if err := json.Unmarshal(contents, &pc.snapshot); err != nil {
			return fmt.Errorf("failed to read snapshot %s: %w", pc.snapshotPath, err)
		}
	}

	return nil
}

func runPush(args []string) error {
	cexts := []config.Configurer{
		&bazelConfigurer{},
		&pushConfigurer{},
	}
	c, err := newConfiguration("push", args, cexts, pushUsage)
	if err != nil {
		return err
	}

	ctx := context.Background()

	pc := getPushConfig(c)

	// log for debugging
	log.Printf("name:      %s", pc.name)
	log.Printf("workspace: %s", pc.workspacePath)
	log.Printf("storage:    %s", pc.storageURL)

	obj, err := push(ctx, pc)
	if err != nil {
		return err
	}

	contentLenght, isOk := obj.GetContentLength()
		if !isOk {
			log.Printf("failed to get contentLenght of pushed snapshot: %s", obj.Path)
		}

	log.Printf("pushed snapshot of %d bytes: %s", contentLenght, obj.Path)

	return nil
}

func push(ctx context.Context, pc *pushConfig) (*types.Object, error) {
	if pc.snapshot == nil {
		return nil, fmt.Errorf("no snapshot specified")
	}

	snapshotBytes, err := json.MarshalIndent(pc.snapshot, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	store, err := storage.NewStorage(pc.storageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	location := fmt.Sprintf("%s/snapshots/%s.json", pc.workspaceName, pc.name)
	reader := bytes.NewReader(snapshotBytes)
	if _, err := store.WriteWithContext(ctx, location, reader, int64(reader.Len())); err != nil {
		return nil, fmt.Errorf("failed to write to bucket file: %w", err)
	}

	obj, err := store.StatWithContext(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to get object details: %w", err)
	}
	return obj, nil
}

func pushUsage(fs *flag.FlagSet) {
	fmt.Fprint(os.Stderr, `usage: push --name=<name> --snapshot-path=<path>

Pushes a snapshot specified by path. Name defaults to the current git HEAD,
or can optionally be specified.

FLAGS:
`)
	fs.PrintDefaults()
}
