package bazel

import (
	"errors"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBuildEventsFile(t *testing.T) {
	input := `
{"id": {"targetCompleted":{"label":"//foo:bar"}}}
{"id": {"targetCompleted":{"label":"//foo:baz"}}}
{"id": {"targetCompleted":{"label":"//foo:qux"}}}
`
	var got []BuildEventOutput
	for ev, err := range ParseBuildEventsFile(strings.NewReader(input)) {
		require.NoError(t, err)
		got = append(got, ev)
	}

	require.Len(t, got, 3)
	assert.Equal(t, "//foo:bar", got[0].ID.TargetCompleted.Label)
	assert.Equal(t, "//foo:baz", got[1].ID.TargetCompleted.Label)
	assert.Equal(t, "//foo:qux", got[2].ID.TargetCompleted.Label)
}

func TestParseBuildEventsFile_errors(t *testing.T) {
	t.Run("invalid JSON", func(t *testing.T) {
		input := `{"id": {"targetCompleted":`
		var err error
		for _, err = range ParseBuildEventsFile(strings.NewReader(input)) {
		}
		require.Error(t, err)
		assert.ErrorContains(t, err, "error parsing build event file")
	})

	t.Run("read error", func(t *testing.T) {
		giveErr := errors.New("great sadness")
		var err error
		for _, err = range ParseBuildEventsFile(iotest.ErrReader(giveErr)) {
		}
		require.Error(t, err)
		assert.ErrorIs(t, err, giveErr)
	})
}
