/* Copyright 2022 Cognite AS */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/pusher"
)

type pushCmd struct {
	name          string
	snapshotPath  string
	workspacePath string

	snapshot *models.Snapshot

	storageUrl string

	cmd *cobra.Command
}

func newPushCmd() *pushCmd {
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push snapshot",
		Long: `Pushes a snapshot specified by path. Name defaults to the current git HEAD,
or can optionally be specified.`,
	}

	pc := &pushCmd{
		cmd: cmd,
	}

	cmd.PersistentFlags().StringVar(&pc.name, "name", "", "snapshot name (defaults to HEAD git sha)")
	cmd.PersistentFlags().StringVar(&pc.snapshotPath, "snapshot-path", "", "path to snapshot to be pushed")
	cmd.PersistentFlags().StringVar(&pc.workspacePath, "workspace-path", "", "workspace path")

	cmd.RunE = pc.runPush

	return pc
}

func (pc *pushCmd) checkArgs(args []string) error {
	// If name is not set, find name from git head
	if pc.name == "" {
		head, err := getGitHead(pc.workspacePath)
		if err != nil {
			return fmt.Errorf("failed to find name from git: %w", err)
		}
		pc.name = head
	}

	storageUrl, err := pc.cmd.Flags().GetString("storage-url")
	if err != nil {
		return err
	}
	pc.storageUrl = storageUrl

	// Read the manifest
	if pc.snapshot == nil && pc.snapshotPath != "" {
		// If it's a relative path, assume workspace-relative. The command is
		// probably run with `bazel run`, and we don't know from where.
		if !path.IsAbs(pc.snapshotPath) {
			pc.snapshotPath = path.Join(pc.workspacePath, pc.snapshotPath)
		}

		log.Println("reading snapshot from", pc.snapshotPath)
		pc.snapshot = &models.Snapshot{}
		contents, err := os.ReadFile(pc.snapshotPath)
		if err != nil {
			return fmt.Errorf("failed to read snapshot path: %w", err)
		}
		if err := json.Unmarshal(contents, &pc.snapshot); err != nil {
			return fmt.Errorf("failed to read snapshot %s: %w", pc.snapshotPath, err)
		}
	}

	return nil
}

func (pc *pushCmd) runPush(cmd *cobra.Command, args []string) error {
	err := pc.checkArgs(args)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// log for debugging
	log.Printf("name:      %s", pc.name)
	log.Printf("workspace: %s", pc.workspacePath)
	log.Printf("storage:    %s", pc.storageUrl)

	pushArgs := pusher.PushArgs{
		Name:       pc.name,
		StorageUrl: pc.storageUrl,
		Snapshot:   pc.snapshot,
	}
	obj, err := pusher.NewPusher().Push(ctx, &pushArgs)
	if err != nil {
		return err
	}

	contentLength := obj.ContentLength
	log.Printf("pushed snapshot of %d bytes: %s", contentLength, obj.Path)

	return nil
}
