package collecter

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/bazel"
	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamedSetOfFiles(t *testing.T) {
	type fileSetEvent struct {
		ID    string
		Files bazel.NamedSetOfFiles
	}

	tests := []struct {
		name string
		give []fileSetEvent
		want map[string][]string // id -> []files
	}{
		{
			name: "UnknownFileSet",
			want: map[string][]string{
				"0": {},
			},
		},
		{
			name: "FileSetWithFiles",
			give: []fileSetEvent{
				{
					ID: "0",
					Files: bazel.NamedSetOfFiles{
						Files: []bazel.NamedSetOfFilesFile{
							{Name: "a", URI: "file:///a"},
							{Name: "b", URI: "file:///b"},
						},
					},
				},
			},
			want: map[string][]string{
				"0": {"file:///a", "file:///b"},
			},
		},
		{
			name: "FileSetWithNestedFileSets",
			give: []fileSetEvent{
				{
					ID: "0",
					Files: bazel.NamedSetOfFiles{
						Files: []bazel.NamedSetOfFilesFile{
							{Name: "a", URI: "file:///a"},
							{Name: "b", URI: "file:///b"},
						},
					},
				},
				{
					ID: "1",
					Files: bazel.NamedSetOfFiles{
						Files: []bazel.NamedSetOfFilesFile{
							{Name: "c", URI: "file:///c"},
						},
						FileSets: []bazel.NamedSetOfFilesFileSet{
							{ID: "0"},
						},
					},
				},
				{
					ID: "2",
					Files: bazel.NamedSetOfFiles{
						Files: []bazel.NamedSetOfFilesFile{
							{Name: "d", URI: "file:///d"},
						},
					},
				},
				{
					ID: "3",
					Files: bazel.NamedSetOfFiles{
						FileSets: []bazel.NamedSetOfFilesFileSet{
							{ID: "1"},
							{ID: "2"},
						},
					},
				},
			},
			want: map[string][]string{
				"0": {"file:///a", "file:///b"},
				"1": {"file:///c", "file:///a", "file:///b"},
				"2": {"file:///d"},
				"3": {"file:///c", "file:///a", "file:///b", "file:///d"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := make(namedSetsOfFiles)
			for _, ev := range tt.give {
				files.Put(ev.ID, ev.Files)
			}

			for id, wantFiles := range tt.want {
				got := slices.Collect(files.ByID(id))

				// It's a set so file order is not guaranteed.
				assert.ElementsMatch(t, wantFiles, got, "files for id %q", id)
			}
		})
	}
}

// writeFakeBazel writes a shell script that stands in for bazel. It emits
// build events pointing at a tracker file. The tracker is "evicted" (absent)
// until the script is invoked with --noremote_accept_cached, at which point
// it is re-created, like a forced rebuild re-uploading to the CAS.
func writeFakeBazel(t *testing.T, dir, tracker string) string {
	t.Helper()
	script := `#!/bin/sh
echo "$@" >> "` + dir + `/invocations"
for a in "$@"; do
  case "$a" in
    --build_event_json_file=*) bep="${a#--build_event_json_file=}" ;;
    --noremote_accept_cached) echo '{"digest":"abc"}' > "` + tracker + `" ;;
  esac
done
cat > "$bep" <<'JSON'
{"id":{"namedSet":{"id":"0"}},"namedSetOfFiles":{"files":[{"name":"t.json","uri":"file://` + tracker + `"}]}}
{"id":{"targetCompleted":{"label":"//pkg:target"}},"completed":{"success":true,"outputGroup":[{"name":"change_track_files","fileSets":[{"id":"0"}]}]}}
JSON
`
	path := filepath.Join(dir, "fakebazel")
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	return path
}

func TestCollect_RetriesOnCacheEviction(t *testing.T) {
	tests := []struct {
		name        string
		retries     int
		wantErr     bool
		wantInvokes int
	}{
		{name: "RetrySucceeds", retries: 2, wantInvokes: 2},
		{name: "NoRetriesFails", retries: 0, wantErr: true, wantInvokes: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tracker := filepath.Join(dir, "tracker.json") // initially missing
			bazel := writeFakeBazel(t, dir, tracker)

			snap, err := NewCollecter().Collect(&CollectArgs{
				BazelPath:            bazel,
				BazelWorkspacePath:   dir,
				BazelExpression:      "//...",
				NoPrint:              true,
				CacheEvictionRetries: tt.retries,
			})

			inv, rerr := os.ReadFile(filepath.Join(dir, "invocations"))
			require.NoError(t, rerr)
			assert.Equal(t, tt.wantInvokes, strings.Count(string(inv), "\n"))

			if tt.wantErr {
				require.ErrorIs(t, err, cache.ErrUnavailable)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "abc", snap.Labels["//pkg:target"].Digest)
			assert.Contains(t, string(inv), "--noremote_accept_cached")
		})
	}
}
