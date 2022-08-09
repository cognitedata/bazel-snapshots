package digester

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"

	"github.com/cognitedata/bazel-snapshots/snapshots/go/pkg/models"
)

type digester struct{}

func NewDigester() *digester {
	return &digester{}
}

func (d *digester) Digest(inPaths, run, tags []string, outPath string) error {
	// sort the input files for more stability
	sort.Strings(inPaths)

	ct := &models.Tracker{
		Run:  run,
		Tags: tags,
	}

	h := sha256.New()
	for _, input := range inPaths {
		// add the filename
		h.Write([]byte(path.Base(input)))

		// add the contents of the file
		f, err := os.Open(input)
		if err != nil {
			return err
		}

		if _, err := io.Copy(h, f); err != nil {
			return fmt.Errorf("failed to digest %s: %w", input, err)
		}
	}

	ct.Digest = fmt.Sprintf("%x", h.Sum(nil))

	content, err := json.Marshal(ct)
	if err != nil {
		return fmt.Errorf("failed to render json file: %w", err)
	}

	return ioutil.WriteFile(outPath, content, 0644)
}
