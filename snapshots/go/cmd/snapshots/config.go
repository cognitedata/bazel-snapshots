/* Copyright 2022 Cognite AS */

package main

import (
	"fmt"
	"os"
	"os/exec"

	flag "github.com/spf13/pflag"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/config"
)

// commonConfig holds common configuration for all commands; args which can
// always be passed.
type commonConfig struct {
	storageURL    string
	gcsBucket     string
	workspaceName string
	verbose       bool
}

const commonName = "_common"

type commonConfigurer struct{}

func getCommonConfig(c *config.Config) *commonConfig {
	return c.Exts[commonName].(*commonConfig)
}

func (*commonConfigurer) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	cc := &commonConfig{}
	c.Exts[commonName] = cc
	fs.StringVar(&cc.storageURL, "storage-url", "", "full orl of the storage")
	fs.StringVar(&cc.gcsBucket, "gcs-bucket", "", "gcs bucket to store snapshots")
	fs.StringVar(&cc.workspaceName, "workspace-name", "", "name of bazel workspace")
	fs.BoolVar(&cc.verbose, "verbose", false, "verbose output")
}

func (*commonConfigurer) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	return nil
}

// bazelConfig holds values useful for interacting with Bazel
type bazelConfig struct {
	bazelPath     string
	workspacePath string
}

const bazelName = "_bazel"

type bazelConfigurer struct{}

func getBazelConfig(c *config.Config) *bazelConfig {
	return c.Exts[bazelName].(*bazelConfig)
}

func (*bazelConfigurer) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	bc := &bazelConfig{}
	c.Exts[bazelName] = bc
	fs.StringVar(&bc.bazelPath, "bazel_path", "", "bazel path (defaults to lookup)")
	fs.StringVar(&bc.workspacePath, "workspace_path", "", "workspace path (defaults to BUILD_WORKSPACE_DIRECTORY)")
}

func (*bazelConfigurer) CheckFlags(fs *flag.FlagSet, c *config.Config) (err error) {
	bc := getBazelConfig(c)

	if bc.bazelPath == "" {
		bc.bazelPath, err = exec.LookPath("bazel")
		if err != nil {
			return err
		}
	}

	if bc.workspacePath == "" {
		if wsDir := os.Getenv("BUILD_WORKSPACE_DIRECTORY"); wsDir != "" {
			bc.workspacePath = wsDir
		} else {
			return fmt.Errorf("-workspace_path not specified and BUILD_WORKSPACE_DIRECTORY not set")
		}
	}

	return
}
