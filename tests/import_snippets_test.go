package tests

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"

	bazeltools "github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var integrityHashRe = regexp.MustCompile(`integrity = "(sha256-[a-zA-Z0-9+/=]+)"`)

func TestImportSnippets(t *testing.T) {
	const dummyContents = "dummy archive content"

	archive := filepath.Join(t.TempDir(), "snippets.tar.gz")
	require.NoError(t, os.WriteFile(archive, []byte(dummyContents), 0o644))

	script, err := bazeltools.Runfile(".github/scripts/import_snippets.sh")
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	defer func() {
		// Surface stderr for debugging if the test fails.
		if t.Failed() {
			t.Logf("stderr:\n%s", stderr.String())
		}
	}()

	cmd := exec.Command("bash", script, archive)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(), "GITHUB_REF_NAME=v1.2.3")
	require.NoError(t, cmd.Run())
	require.Empty(t, stderr.String(), "stderr should be empty")

	// Verify that the integrity hash is valid.
	matches := integrityHashRe.FindStringSubmatch(stdout.String())
	require.Len(t, matches, 2, "integrity hash should be found")
	gotIntegrityHash := matches[1]

	wantSHA := sha256.Sum256([]byte(dummyContents))
	wantIntegrityHash := "sha256-" + base64.StdEncoding.EncodeToString(wantSHA[:])

	assert.Equal(t, wantIntegrityHash, gotIntegrityHash, "integrity hash should match expected value")
}
