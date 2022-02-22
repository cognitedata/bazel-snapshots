package bazel

import (
	"context"
	"fmt"
	"net"
	"testing"

	bazeltools "github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/bytestream"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestFileBazelCache(t *testing.T) {
	c := &FileBazelCache{}
	ctx := context.Background()

	var contents []byte
	var err error

	// invalid scheme
	contents, err = c.Read(ctx, "no-scheme")
	require.ErrorIs(t, err, ErrScheme)
	require.Nil(t, contents)

	// wrong scheme
	contents, err = c.Read(ctx, "http://some-file")
	require.ErrorIs(t, err, ErrScheme)
	require.Nil(t, contents)

	// read this file
	thisFile, err := bazeltools.Runfile("snapshots/go/pkg/bazel/cache_test.go")
	require.Nil(t, err)
	contents, err = c.Read(ctx, fmt.Sprintf("file://%s", thisFile))
	require.Nil(t, err)
	require.Contains(t, string(contents), "literally anything goes here because we're reading this file")
	require.NotNil(t, contents)
}

func TestDelegatingBazelCache(t *testing.T) {
	c := &DelegatingBazelCache{
		caches: map[string]BazelCache{
			"file": &FileBazelCache{},
		},
	}
	ctx := context.Background()

	var contents []byte
	var err error

	// invalid scheme
	contents, err = c.Read(ctx, "no-scheme")
	require.ErrorIs(t, err, ErrScheme)
	require.Nil(t, contents)

	// wrong scheme
	contents, err = c.Read(ctx, "wrongscheme://some-uri")
	require.ErrorIs(t, err, ErrScheme)
	require.Nil(t, contents)

	// file not found
	contents, err = c.Read(ctx, "file:///doesnt-exist")
	require.ErrorIs(t, err, ErrUnavailable)
	require.Nil(t, contents)

	// delegate to file
	thisFile, err := bazeltools.Runfile("snapshots/go/pkg/bazel/cache_test.go")
	require.Nil(t, err)
	contents, err = c.Read(ctx, fmt.Sprintf("file://%s", thisFile))
	require.Nil(t, err)
	require.NotNil(t, contents)
}

type mockByteStreamServer struct {
	bytestream.UnimplementedByteStreamServer
}

func (s *mockByteStreamServer) Read(req *bytestream.ReadRequest, stream bytestream.ByteStream_ReadServer) error {
	if req.GetResourceName() != "bytestream://bufnet/cache-key" {
		return status.Errorf(codes.NotFound, "wrong cache key: %s", req.GetResourceName())
	}

	message := &bytestream.ReadResponse{
		Data: []byte("hello world"),
	}
	if err := stream.Send(message); err != nil {
		return err
	}

	return nil
}

func TestRemoteBazelCache(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	bytestream.RegisterByteStreamServer(s, &mockByteStreamServer{})

	bufDialer := func(ctx context.Context, address string) (net.Conn, error) {
		return lis.Dial()
	}
	go func() {
		if err := s.Serve(lis); err != nil {
			require.Nil(t, err)
		}
	}()

	c := &RemoteBazelCache{
		clients: make(map[string]bytestream.ByteStreamClient),
		DialOptions: []grpc.DialOption{
			grpc.WithInsecure(),
			grpc.WithContextDialer(bufDialer),
		},
	}
	ctx := context.Background()

	var contents []byte
	var err error

	// invalid scheme
	contents, err = c.Read(ctx, "no-scheme")
	require.ErrorIs(t, err, ErrScheme)
	require.Nil(t, contents)

	// wrong scheme
	contents, err = c.Read(ctx, "file://some-file")
	require.ErrorIs(t, err, ErrScheme)
	require.Nil(t, contents)

	// wrong cache key
	contents, err = c.Read(ctx, "bytestream://bufnet/wrong-key")
	require.ErrorIs(t, err, ErrUnavailable)
	require.Nil(t, contents)

	// good request
	contents, err = c.Read(ctx, "bytestream://bufnet/cache-key")
	require.Nil(t, err)
	require.Equal(t, string(contents), "hello world")
}
