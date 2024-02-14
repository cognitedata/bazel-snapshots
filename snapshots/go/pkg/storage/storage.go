/* Copyright 2022 Cognite AS */

package storage

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	awscfg "github.com/aws/aws-sdk-go-v2/config"
	_ "go.beyondstorage.io/services/gcs/v3"
	_ "go.beyondstorage.io/services/s3/v3"
	"go.beyondstorage.io/v5/pairs"
	"go.beyondstorage.io/v5/services"
	"go.beyondstorage.io/v5/types"
)

var IteratorDone = types.IterateDone

func NewStorage(storageURL string) (types.Storager, error) {
	ctx := context.Background()

	u, err := url.Parse(storageURL)
	if err != nil {
		return nil, err
	}

	values := u.Query()

	// for s3, credentials can't be set from URL, so we handle it specially
	if u.Scheme == "s3" {

		// get the default credentials
		config, err := awscfg.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, err
		}

		name := u.Hostname()

		// make sure path ends with a '/'
		workdir := u.Path
		if !strings.HasSuffix(workdir, "/") {
			workdir = fmt.Sprintf("%s/", workdir)
		}

		// read credentials from query or fall back to default
		creds := values.Get("credentials")
		if creds == "" {
			c, err := config.Credentials.Retrieve(ctx)
			if err != nil {
				return nil, err
			}
			creds = fmt.Sprintf("hmac:%s:%s", c.AccessKeyID, c.SecretAccessKey)
		}

		// read location from query or fall back to default config
		location := values.Get("location")
		if location == "" {
			location = config.Region
		}

		return services.NewStorager(
			"s3",
			pairs.WithName(name),
			pairs.WithCredential(creds),
			pairs.WithLocation(location),
			pairs.WithWorkDir(workdir),
		)
	}

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
