/* Copyright 2022 Cognite AS */

package storage

import (
	"github.com/graymeta/stow"
	// support Google storage
	gcs "github.com/graymeta/stow/google"
)

// Config is Google Cloud Storage config.
func Config(json, projectID string) stow.ConfigMap {
	return stow.ConfigMap{
		gcs.ConfigJSON:      json,
		gcs.ConfigProjectId: projectID,
	}
}
