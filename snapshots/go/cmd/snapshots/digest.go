/* Copyright 2022 Cognite AS */

package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/digester"
)

type digestCmd struct {
	inPaths     []string
	run         []string
	tags        []string
	outPath     string
	inPathsFile string

	cmd *cobra.Command
}

func newDigestCmd() *digestCmd {
	cmd := &cobra.Command{
		Use:   "digest",
		Short: "Digest snapshots",
		Long: `Writes a digest of the infiles to an outfile. Stable on infile order. Includes
the filenames in the digest. Outputs a JSON file containing a digest of the
files, plus metadata determined by other flags.`,
	}

	dc := &digestCmd{
		cmd: cmd,
	}

	cmd.PersistentFlags().StringArrayVar(&dc.inPaths, "in-paths", nil, "Input files to read")
	cmd.PersistentFlags().StringArrayVar(&dc.run, "run", nil, "Run")
	cmd.PersistentFlags().StringArrayVar(&dc.tags, "tag", nil, "Tags")
	cmd.PersistentFlags().StringVar(&dc.outPath, "out", "", "Output path")
	cmd.PersistentFlags().StringVar(&dc.inPathsFile, "inputs-file", "", "File containing input paths to read, one per line")

	cmd.RunE = dc.runDigest

	return dc
}

func (dc *digestCmd) checkArgs(args []string) error {
	dc.inPaths = args
	if dc.inPathsFile != "" {
		f, err := os.Open(dc.inPathsFile)
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		defer func() { _ = f.Close() }()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				dc.inPaths = append(dc.inPaths, line)
			}
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("read file: %w", err)
		}
	}

	if len(dc.inPaths) == 0 {
		return fmt.Errorf("need at least one path to digest")
	}

	return nil
}

func (dc *digestCmd) runDigest(cmd *cobra.Command, args []string) error {
	err := dc.checkArgs(args)
	if err != nil {
		return err
	}

	digestArgs := digester.DigestArgs{
		InPaths: dc.inPaths,
		Run:     dc.run,
		Tags:    dc.tags,
		OutPath: dc.outPath,
	}
	return digester.NewDigester().Digest(&digestArgs)
}
