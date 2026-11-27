package collecter

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/bytestream"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// evictingCAS is a fake remote CAS (gRPC ByteStream) whose blob is missing,
// like after an eviction, until the "rebuilt" marker file exists.
type evictingCAS struct {
	bytestream.UnimplementedByteStreamServer
	rebuilt string
}

func (s *evictingCAS) Read(_ *bytestream.ReadRequest, stream bytestream.ByteStream_ReadServer) error {
	if _, err := os.Stat(s.rebuilt); err != nil {
		return status.Error(codes.NotFound, "blob evicted")
	}
	return stream.Send(&bytestream.ReadResponse{Data: []byte(`{"digest":"abc"}`)})
}

// TestCollect_EndToEnd_CASEviction runs Collect against a fake bazel binary
// and a real gRPC ByteStream server on localhost, with the tracker blob
// initially evicted. It exercises the whole path: bazel invocation, build
// events, gRPC read, NotFound -> ErrUnavailable, forced rebuild, retry.
//
// Run locally with:
//
//	bazel test //snapshots/go/pkg/collecter:collecter_test --test_output=streamed
//
// or, without Bazel: go test ./snapshots/go/pkg/collecter -run EndToEnd -v
func TestCollect_EndToEnd_CASEviction(t *testing.T) {
	dir := t.TempDir()
	rebuilt := filepath.Join(dir, "rebuilt")

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	bytestream.RegisterByteStreamServer(srv, &evictingCAS{rebuilt: rebuilt})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	uri := "bytestream://" + lis.Addr().String() + "/blobs/deadbeef/16"
	script := `#!/bin/sh
echo "$@" >> "` + dir + `/invocations"
for a in "$@"; do
  case "$a" in
    --build_event_json_file=*) bep="${a#--build_event_json_file=}" ;;
    --noremote_accept_cached) touch "` + rebuilt + `" ;;
  esac
done
cat > "$bep" <<'JSON'
{"id":{"namedSet":{"id":"0"}},"namedSetOfFiles":{"files":[{"name":"t.json","uri":"` + uri + `"}]}}
{"id":{"targetCompleted":{"label":"//pkg:target"}},"completed":{"success":true,"outputGroup":[{"name":"change_track_files","fileSets":[{"id":"0"}]}]}}
JSON
`
	bazel := filepath.Join(dir, "fakebazel")
	require.NoError(t, os.WriteFile(bazel, []byte(script), 0o755))

	collect := func(eventsPath string) (*models.Snapshot, error) {
		return NewCollecter().Collect(&CollectArgs{
			BazelPath:            bazel,
			BazelWorkspacePath:   dir,
			BazelExpression:      "//...",
			BazelBuildEventsPath: eventsPath,
			BazelCacheGrpcs:      false, // plaintext gRPC to the local fake
			NoPrint:              true,
			CacheEvictionRetries: 2,
		})
	}

	snap, err := collect("")
	require.NoError(t, err)
	assert.Equal(t, "abc", snap.Labels["//pkg:target"].Digest)

	inv, err := os.ReadFile(filepath.Join(dir, "invocations"))
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(inv)), "\n")
	require.Len(t, lines, 2)
	assert.NotContains(t, lines[0], "--noremote_accept_cached")
	assert.Contains(t, lines[1], "--noremote_accept_cached")

	// Same, when the caller ran the build itself and passes the events file
	// (how the infrastructure repo uses it): the rebuild rewrites that file.
	require.NoError(t, os.Remove(rebuilt)) // blob is evicted again
	events := filepath.Join(dir, "build-events.json")
	_, err = exec.Command(bazel, "build", "--build_event_json_file="+events).CombinedOutput()
	require.NoError(t, err)
	require.NoError(t, os.Remove(filepath.Join(dir, "invocations")))

	snap, err = collect(events)
	require.NoError(t, err)
	assert.Equal(t, "abc", snap.Labels["//pkg:target"].Digest)
	inv, err = os.ReadFile(filepath.Join(dir, "invocations"))
	require.NoError(t, err)
	assert.Contains(t, string(inv), "--noremote_accept_cached")
}
