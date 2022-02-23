package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"

	flag "github.com/spf13/pflag"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/config"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
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

	return digest(dc)
}

func digest(dc *digestConfig) error {
	// sort the input files for more stability
	sort.Strings(dc.inPaths)

	ct := &models.Tracker{
		Run:  dc.run,
		Tags: dc.tags,
	}

	h := sha256.New()
	for _, input := range dc.inPaths {
		// add the filename
		h.Write([]byte(path.Base(input)))

		// add the contents of the file
		f, err := os.Open(input)
		if err != nil {
			return err
		}

		if _, err := io.Copy(h, f); err != nil {
			return fmt.Errorf("failed to digest %s: %w", input, err)
		}
	}

	ct.Digest = fmt.Sprintf("%x", h.Sum(nil))

	content, err := json.Marshal(ct)
	if err != nil {
		return fmt.Errorf("failed to render json file: %w", err)
	}

	return ioutil.WriteFile(dc.outPath, content, 0644)
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
