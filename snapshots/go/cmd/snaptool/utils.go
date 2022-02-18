package main

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	flag "github.com/spf13/pflag"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/config"
)

// newConfiguration creates configs for a given set of configurers
func newConfiguration(name string, args []string, cexts []config.Configurer, usage func(*flag.FlagSet)) (*config.Config, error) {
	c := config.New()

	cexts = append([]config.Configurer{&commonConfigurer{}}, cexts...)

	fs := flag.NewFlagSet("snaptool", flag.ContinueOnError)
	fs.Usage = func() {}
	for _, cext := range cexts {
		cext.RegisterFlags(fs, name, c)
	}
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			usage(fs)
			return nil, err
		}
		return nil, fmt.Errorf("Try -help for more information: %w", err)
	}
	for _, cext := range cexts {
		if err := cext.CheckFlags(fs, c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func getGitHead(path string) (string, error) {
	r, err := git.PlainOpen(path)
	if err != nil {
		return "", fmt.Errorf("failed to open git repo %s: %w", path, err)
	}
	head, err := r.Head()
	if err != nil {
		return "", fmt.Errorf("failed to find git head: %w", err)
	}
	return head.Hash().String(), nil
}
