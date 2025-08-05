/* Copyright 2022 Cognite AS */

package storage

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/url"
	"os"
	"strings"

	awscfg "github.com/aws/aws-sdk-go-v2/config"
	_ "go.beyondstorage.io/services/fs/v4"
	_ "go.beyondstorage.io/services/gcs/v3"
	_ "go.beyondstorage.io/services/s3/v3"
	"go.beyondstorage.io/v5/pairs"
	"go.beyondstorage.io/v5/services"
	"go.beyondstorage.io/v5/types"
)

// Storage stores files in a cloud storage bucket or a local file system.
type Storage struct {
	bucket types.Storager
}

func NewStorage(storageURL string) (*Storage, error) {
	storager, err := newStorager(storageURL)
	if err != nil {
		return nil, fmt.Errorf("new storager: %w", err)
	}

	return &Storage{
		bucket: storager,
	}, nil
}

func newStorager(storageURL string) (types.Storager, error) {
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

	if u.Scheme == "file" {
		// Hack: "fs" == "file".
		// Will be deleted when we switch to gocloud.dev.
		u.Scheme = "fs"
	}

	u.RawQuery = values.Encode()

	return services.NewStoragerFromString(u.String())
}

// ErrNotExist indicates that a requested path does not exist.
var ErrNotExist = os.ErrNotExist

// ReadAll reads the entire content of a file at the specified path.
// Returns [ErrNotExist] if the file does not exist.
func (s *Storage) ReadAll(ctx context.Context, path string) ([]byte, error) {
	var buf bytes.Buffer
	_, err := s.bucket.ReadWithContext(ctx, path, &buf)
	if err != nil {
		if errors.Is(err, services.ErrObjectNotExist) {
			return nil, ErrNotExist
		}
		return nil, fmt.Errorf("read all: %w", err)
	}
	return buf.Bytes(), nil
}

// WriteAll writes the entire content of a file at the specified path.
func (s *Storage) WriteAll(ctx context.Context, path string, bs []byte) error {
	_, err := s.bucket.WriteWithContext(ctx, path, bytes.NewReader(bs), int64(len(bs)))
	return err
}

// ObjectMetadata contains metadata about an object in the storage.
type ObjectMetadata struct {
	// Path is the path to the object in the storage
	// relative to the bucket root.
	Path string

	// ContentLength is the size of the object in bytes.
	ContentLength int64
}

// Stat inspects an object in the storage and returns its metadata.
// Returns [ErrNotExist] if the object does not exist.
func (s *Storage) Stat(ctx context.Context, path string) (*ObjectMetadata, error) {
	attrs, err := s.bucket.StatWithContext(ctx, path)
	if err != nil {
		if errors.Is(err, services.ErrObjectNotExist) {
			return nil, ErrNotExist
		}
		return nil, fmt.Errorf("stat: %w", err)
	}

	contentLength, ok := attrs.GetContentLength()
	if !ok {
		return nil, fmt.Errorf("stat: content length not available")
	}

	return &ObjectMetadata{
		Path:          path,
		ContentLength: contentLength,
	}, nil
}

// ReadInto reads the content of a file at the specified path
// and writes it to the provided writer.
//
// Returns the number of bytes written and an error if any.
// Returns [ErrNotExist] if the file does not exist.
func (s *Storage) ReadInto(ctx context.Context, path string, w io.Writer) (int64, error) {
	n, err := s.bucket.ReadWithContext(ctx, path, w)
	if err != nil {
		if errors.Is(err, services.ErrObjectNotExist) {
			return 0, ErrNotExist
		}
		return 0, fmt.Errorf("new reader: %w", err)
	}
	return n, nil
}

// ListObject is an object in a bucket list iteration.
type ListObject struct {
	Path string
}

// List returns an iterator over objects in the storage
// with the specified prefix.
func (s *Storage) List(ctx context.Context, prefix string) iter.Seq2[ListObject, error] {
	return func(yield func(ListObject, error) bool) {
		it, err := s.bucket.ListWithContext(ctx, prefix)
		if err != nil {
			yield(ListObject{}, fmt.Errorf("list: %w", err))
			return
		}

		for {
			attrs, err := it.Next()
			if err != nil {
				if errors.Is(err, types.IterateDone) {
					return
				}

				yield(ListObject{}, fmt.Errorf("list: %w", err))
				return
			}

			if !yield(ListObject{Path: attrs.Path}, nil) {
				return
			}
		}
	}
}
