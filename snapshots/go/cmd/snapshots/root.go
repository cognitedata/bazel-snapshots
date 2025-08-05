/* Copyright 2022 Cognite AS */

package main

import (
	"github.com/spf13/cobra"
)

type rootCmd struct {
	storageURL string
	verbose    bool

	cmd *cobra.Command
}

func newRootCmd() *rootCmd {
	cmd := &cobra.Command{
		Use:   "snapshots",
		Short: "snapshots - a tool for creating and interacting with Snapshots for Bazel",
		Long: `Snapshots is a tool for creating and interacting with Snapshots for Bazel.
	These snapshots are summaries of the outputs of a set of Bazel targets, and
	can be used to check whether a target has changed.`,
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	cmd.AddCommand(newCollectCmd().cmd)
	cmd.AddCommand(newDiffCmd().cmd)
	cmd.AddCommand(newDigestCmd().cmd)
	cmd.AddCommand(newGetCmd().cmd)
	cmd.AddCommand(newPushCmd().cmd)
	cmd.AddCommand(newTagCmd().cmd)

	rc := &rootCmd{
		cmd: cmd,
	}

	cmd.PersistentFlags().StringVarP(&rc.storageURL, "storage-url", "s", "", "Full URL of the storage")
	cmd.PersistentFlags().BoolVarP(&rc.verbose, "verbose", "v", false, "Verbose output")

	return rc
}
