/* Copyright 2022 Cognite AS */

package main

import (
	"fmt"
	"os"

	flag "github.com/spf13/pflag"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/config"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/digester"
)

type digestConfig struct {
	inPaths []string
	run     []string
	tags    []string
	outPath string
}

const digestName = "_digest"

func getDigestConfig(c *config.Config) *digestConfig {
	return c.Exts[digestName].(*digestConfig)
}

type digestConfigurer struct{}

func (dcc *digestConfigurer) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	dc := &digestConfig{}
	c.Exts[digestName] = dc

	fs.StringVar(&dc.outPath, "out", "", "output file path")
	fs.StringSliceVar(&dc.run, "run", []string{}, "run labels (repeated)")
	fs.StringSliceVar(&dc.tags, "tag", []string{}, "tags (repeated)")
}

func (dcc *digestConfigurer) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	dc := getDigestConfig(c)

	dc.inPaths = fs.Args()
	if len(dc.inPaths) == 0 {
		return fmt.Errorf("need at least one path to digest")
	}

	return nil
}

func runDigest(args []string) (err error) {
	cexts := []config.Configurer{
		&digestConfigurer{},
	}
	c, err := newConfiguration("digest", args, cexts, digestUsage)
	if err != nil {
		return err
	}

	dc := getDigestConfig(c)

	digestArgs := digester.DigestArgs{
		InPaths: dc.inPaths,
		Run:     dc.run,
		Tags:    dc.tags,
		OutPath: dc.outPath,
	}
	return digester.NewDigester().Digest(&digestArgs)
}

func digestUsage(fs *flag.FlagSet) {
	fmt.Fprint(os.Stderr, `usage: digest -outfile <path> <inpath> [<inpath> ...]

Writes a digest of the infiles to an outfile. Stable on infile order. Includes
the filenames in the digest. Outputs a JSON file containing a digest of the
files, plus metadata determined by other flags.

FLAGS:
`)
	fs.PrintDefaults()
}
