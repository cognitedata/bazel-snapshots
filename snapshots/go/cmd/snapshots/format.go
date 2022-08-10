/* Copyright 2022 Cognite AS */

package main

import (
	"errors"
)

type OutputFormat string

const (
	formatLabel  OutputFormat = "label"
	formatJson   OutputFormat = "json"
	formatPretty OutputFormat = "pretty"
)

// String is used both by fmt.Print and by Cobra in help text
func (e *OutputFormat) String() string {
	return string(*e)
}

// Set must have pointer receiver so it doesn't change the value of a copy
func (e *OutputFormat) Set(v string) error {
	switch v {
	case "label", "json", "pretty":
		*e = OutputFormat(v)
		return nil
	default:
		return errors.New(`must be one of "label", "json", or "pretty"`)
	}
}

// Type is only used in help text
func (e *OutputFormat) Type() string {
	return "format"
}
