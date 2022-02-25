/* Copyright 2022 Cognite AS */

// package models defines models used internally in snapshots
package models

import "encoding/json"

type Tracker struct {
	Digest string   `json:"digest"`
	Run    []string `json:"run,omitempty"`
	Tags   []string `json:"tags,omitempty"`
}

type Snapshot struct {
	Labels map[string]*Tracker `json:"labels"`
}

type ChangeType int

func (a ChangeType) String() string {
	switch a {
	case Unchanged:
		return "unchanged"
	case Added:
		return "added"
	case Removed:
		return "removed"
	case Changed:
		return "changed"
	}
	return ""
}

func (a ChangeType) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

const (
	Unchanged ChangeType = iota
	Added
	Removed
	Changed
)

type TrackerChange struct {
	Tracker
	Label      string     `json:"label"`
	ChangeType ChangeType `json:"change"`
}
