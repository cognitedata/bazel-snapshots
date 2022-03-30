/* Copyright 2022 Cognite AS */

package storage

import (
	"fmt"
	"net/url"
	"strings"

	_ "go.beyondstorage.io/services/gcs/v3"
	"go.beyondstorage.io/v5/services"
	"go.beyondstorage.io/v5/types"
)

var IteratorDone = types.IterateDone

func NewStorage(storageURL string) (types.Storager, error) {
	u, err := url.Parse(storageURL)
	if err != nil {
		return nil, err
	}

	values := u.Query()

	// automatically fix the storage URL for common problems
	if u.Scheme == "gcs" {
		// set default value for 'credential' if not set: will use default
		// credentials.
		if values.Get("credential") == "" {
			values.Add("credential", "env")
		}

		// set default value for 'project_id' if not set: will be inferred.
		if values.Get("project_id") == "" {
			values.Add("project_id", "env")
		}

		// make sure path ends with a '/'
		if !strings.HasSuffix(u.Path, "/") {
			u.Path = fmt.Sprintf("%s/", u.Path)
		}
	}

	u.RawQuery = values.Encode()

	return services.NewStoragerFromString(u.String())
}
