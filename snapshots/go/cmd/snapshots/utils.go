/* Copyright 2022 Cognite AS */

package main

import (
	"fmt"
	"os/exec"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/config"
)

// newConfiguration creates configs for a given set of configurers
func newConfiguration(name string, args []string, cexts []config.Configurer, usage func(*flag.FlagSet)) (*config.Config, error) {
	c := config.New()

	cexts = append([]config.Configurer{&commonConfigurer{}}, cexts...)

	fs := flag.NewFlagSet("snapshots", flag.ContinueOnError)
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
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = path

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get name from git: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}
