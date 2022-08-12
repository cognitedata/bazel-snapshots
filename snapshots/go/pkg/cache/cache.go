/* Copyright 2022 Cognite AS */

package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"google.golang.org/genproto/googleapis/bytestream"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	ErrScheme      = errors.New("wrong or invalid scheme requested")
	ErrUnavailable = errors.New("item is not available in cache")
)

// BazelCache allows reading objects directly from Bazel's caches. This might be
// necessary when one needs access to output files, because Bazel might not
// materialize the file on disk if the result is cached.
type BazelCache interface {
	Read(ctx context.Context, secure bool, uri string) ([]byte, error)
}

// DelegatingBazelCache consists of multiple BazelCaches, and uses the one
// appropriate for a given uri by looking at the scheme.
type DelegatingBazelCache struct {
	caches map[string]BazelCache
}

func NewDefaultDelegatingCache(dialOptions ...grpc.DialOption) BazelCache {
	return &DelegatingBazelCache{
		caches: map[string]BazelCache{
			"file": &FileBazelCache{},
			"bytestream": &RemoteBazelCache{
				clients:     make(map[string]bytestream.ByteStreamClient),
				DialOptions: dialOptions,
			},
		},
	}
}

func (c *DelegatingBazelCache) Read(ctx context.Context, secure bool, uri string) ([]byte, error) {
	u, err := url.ParseRequestURI(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse scheme for %s: %w", uri, ErrScheme)
	}

	cache, ok := c.caches[u.Scheme]
	if !ok {
		return nil, fmt.Errorf("unknown scheme %s: %w", u.Scheme, ErrScheme)
	}

	return cache.Read(ctx, secure, uri)
}

// FileBazelCache provides access to cached items with 'file://' uris.
type FileBazelCache struct{}

func (c *FileBazelCache) Read(ctx context.Context, secure bool, uri string) ([]byte, error) {
	u, err := url.ParseRequestURI(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse scheme for %s: %w", uri, ErrScheme)
	}
	if u.Scheme != "file" {
		return nil, fmt.Errorf("expected scheme to be file, not %s: %w", u.Scheme, ErrScheme)
	}

	contents, err := ioutil.ReadFile(u.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file %s does not exist: %w", u.Path, ErrUnavailable)
		}
		return nil, err
	}

	return contents, nil
}

// RemoteBazelCache provides access to cached items with 'bytestream://' uris.
type RemoteBazelCache struct {
	clients     map[string]bytestream.ByteStreamClient
	DialOptions []grpc.DialOption
}

func (c *RemoteBazelCache) Read(ctx context.Context, secure bool, uri string) ([]byte, error) {
	u, err := url.ParseRequestURI(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse scheme for %s: %w", uri, ErrScheme)
	}

	// obtain a client
	client, ok := c.clients[u.Host]
	if !ok {
		conn, err := DialTargetWithOptions(uri, secure, c.DialOptions...)
		if err != nil {
			return nil, fmt.Errorf("failed to dial host %s: %w", u.Host, err)
		}
		client = bytestream.NewByteStreamClient(conn)
		c.clients[u.Host] = client
	}

	req := &bytestream.ReadRequest{
		ResourceName: strings.TrimPrefix(u.RequestURI(), "/"), 
		ReadOffset:   0,
		ReadLimit:    0,
	}

	bsrc, err := client.Read(ctx, req)
	if err != nil {
		return nil, err
	}

	var blob []byte
	for {
		resp, err := bsrc.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			if grpc.Code(err) == codes.NotFound {
				return nil, fmt.Errorf("item %s not found: %w", uri, ErrUnavailable)
			}
			return nil, fmt.Errorf("failed reading cache response for %s: %w", uri, err)
		}
		if resp == nil {
			return nil, fmt.Errorf("got nil response")
		}
		blob = append(blob, resp.Data...)
	}

	return blob, nil
}
