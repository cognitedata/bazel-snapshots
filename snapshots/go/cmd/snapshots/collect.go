/* Copyright 2022 Cognite AS */

package main

import (
	"fmt"
	"log"
	"os"
	"path"

	flag "github.com/spf13/pflag"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/collecter"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/config"
)

type collectConfig struct {
	bazelConfig
	commonConfig
	queryExpression        string
	bazelCacheGRPCInsecure bool
	bazelStderr            bool
	outPath                string
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
	if _, err := collecter.NewCollecter().Collect(
		cc.bazelPath,
		cc.outPath,
		cc.queryExpression,
		cc.workspacePath,
		cc.bazelCacheGRPCInsecure,
		cc.bazelStderr,
		cc.noPrint); err != nil {
		return fmt.Errorf("failed to collect: %w", err)
	}

	return nil
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
