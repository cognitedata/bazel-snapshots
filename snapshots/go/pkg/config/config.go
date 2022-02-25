/* Copyright 2022 Cognite AS */

// package config provides lightweight configuration utilities for Snapshots.
package config

import (
	flag "github.com/spf13/pflag"
)

type Config struct {
	// Exts is a map of "extensions", where individual configurers can add their
	// data.
	Exts map[string]interface{}
}

func New() *Config {
	return &Config{
		Exts: make(map[string]interface{}),
	}
}

type Configurer interface {
	RegisterFlags(fs *flag.FlagSet, cmd string, config *Config)
	CheckFlags(fs *flag.FlagSet, config *Config) error
}
