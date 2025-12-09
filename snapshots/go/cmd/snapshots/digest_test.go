package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDigestCmd_checkArgs_inputsFile(t *testing.T) {
	inputsFile := filepath.Join(t.TempDir(), "inputs.txt")
	require.NoError(t, os.WriteFile(inputsFile, []byte("file1\nfile2\nfile3\n"), 0o644))

	cmd := digestCmd{
		inPathsFile: inputsFile,
	}
	require.NoError(t, cmd.checkArgs([]string{"file4", "file5"}))
	require.Equal(t, []string{
		"file4", "file5", "file1", "file2", "file3",
	}, cmd.inPaths)
}
