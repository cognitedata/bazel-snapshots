/* Copyright 2022 Cognite AS */

// Package storage implements the storage backend for snapshots.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
	"gocloud.dev/gcerrors"
)

// Storage stores files in a cloud storage bucket or a local file system.
type Storage struct {
	bucket *blob.Bucket
}

// NewStorage builds a storage instance from a storage URL.
// Storage URLs can be in the format:
//
//	file:///path/to/local/storage
//	gs://bucket-name/subdir
//	s3://bucket-name/subdir
//
// For backwards compatibility, "gcs://" may be in place of "gs://".
func NewStorage(storageURL string) (*Storage, error) {
	ctx := context.Background()

	storageURL = backwardsCompatibleStorageURL(storageURL)
	bucket, err := blob.OpenBucket(ctx, storageURL)
	if err != nil {
		return nil, fmt.Errorf("open bucket: %w", err)
	}

	return &Storage{
		bucket: bucket,
	}, nil
}

// ErrNotExist indicates that a requested path does not exist.
var ErrNotExist = os.ErrNotExist

// ReadAll reads the entire content of a file at the specified path.
// Returns [ErrNotExist] if the file does not exist.
func (s *Storage) ReadAll(ctx context.Context, path string) ([]byte, error) {
	bs, err := s.bucket.ReadAll(ctx, path)
	if err != nil {
		if gcerrors.Code(err) == gcerrors.NotFound {
			return nil, ErrNotExist
		}
		return nil, fmt.Errorf("read all: %w", err)
	}
	return bs, nil
}

// WriteAll writes the entire content of a file at the specified path.
func (s *Storage) WriteAll(ctx context.Context, path string, bs []byte) error {
	return s.bucket.WriteAll(ctx, path, bs, nil)
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
	attrs, err := s.bucket.Attributes(ctx, path)
	if err != nil {
		if gcerrors.Code(err) == gcerrors.NotFound {
			return nil, ErrNotExist
		}
		return nil, fmt.Errorf("stat: %w", err)
	}

	return &ObjectMetadata{
		Path:          path,
		ContentLength: attrs.Size,
	}, nil
}

// ReadInto reads the content of a file at the specified path
// and writes it to the provided writer.
//
// Returns the number of bytes written and an error if any.
// Returns [ErrNotExist] if the file does not exist.
func (s *Storage) ReadInto(ctx context.Context, path string, w io.Writer) (int64, error) {
	reader, err := s.bucket.NewReader(ctx, path, nil)
	if err != nil {
		if gcerrors.Code(err) == gcerrors.NotFound {
			return 0, ErrNotExist
		}
		return 0, fmt.Errorf("new reader: %w", err)
	}
	defer reader.Close()

	return reader.WriteTo(w)
}

// ListObject is an object in a bucket list iteration.
type ListObject struct {
	Path string
}

// List returns an iterator over objects in the storage
// with the specified prefix.
func (s *Storage) List(ctx context.Context, prefix string) iter.Seq2[ListObject, error] {
	return func(yield func(ListObject, error) bool) {
		it := s.bucket.List(&blob.ListOptions{
			Prefix:    prefix,
			Delimiter: "/",
		})
		for {
			obj, err := it.Next(ctx)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					yield(ListObject{}, fmt.Errorf("list: %w", err))
				}
				return
			}

			if obj.IsDir {
				continue
			}

			listObj := ListObject{Path: obj.Key}
			if !yield(listObj, nil) {
				return
			}
		}
	}
}
