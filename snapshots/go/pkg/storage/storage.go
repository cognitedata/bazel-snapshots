/* Copyright 2022 Cognite AS */

package storage

import (
	"cmp"
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

	// S3 backend for beyondstorage does not support any query parameters
	// so we implement our own scheme around it
	// on top of default AWS credentials.
	if u.Scheme == "s3" {
		config, err := awscfg.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("load default AWS config: %w", err)
		}

		name := u.Hostname() // bucket name

		// Make sure path ends with a '/'
		workdir := u.Path
		if !strings.HasSuffix(workdir, "/") {
			workdir = fmt.Sprintf("%s/", workdir)
		}

		// Credentials may come from the query string
		// or from the default AWS config (in that order).
		creds := values.Get("credentials")
		if creds == "" {
			c, err := config.Credentials.Retrieve(ctx)
			if err != nil {
				return nil, fmt.Errorf("retrieve default AWS credentials: %w", err)
			}
			creds = fmt.Sprintf("hmac:%s:%s", c.AccessKeyID, c.SecretAccessKey)
		}

		// Region comes from the query string or from the default AWS config.
		region := cmp.Or(values.Get("region"), config.Region)

		return services.NewStorager(
			"s3",
			pairs.WithName(name),
			pairs.WithCredential(creds),
			pairs.WithLocation(region),
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
