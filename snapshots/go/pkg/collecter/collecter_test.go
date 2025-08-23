package collecter

import (
	"slices"
	"testing"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/bazel"
	"github.com/stretchr/testify/assert"
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
