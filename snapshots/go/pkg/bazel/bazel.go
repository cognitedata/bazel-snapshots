/* Copyright 2022 Cognite AS */

// package bazel provides a simple Bazel API for the specific functionality
// needed in snapshots.
package bazel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"os/exec"
)

// Client exposes the Bazel CLI.
type Client struct {
	path   string
	ws     string
	stderr io.Writer
}

func NewClient(path, ws string, stderr io.Writer) *Client {
	return &Client{
		path:   path,
		ws:     ws,
		stderr: stderr,
	}
}

func (c *Client) Command(ctx context.Context, args ...string) ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	cmd := exec.CommandContext(ctx, c.path, args...)
	cmd.Stderr = c.stderr
	cmd.Stdout = buf
	cmd.Dir = c.ws

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("bazel command error: %w", err)
	}

	return buf.Bytes(), nil
}

func (c *Client) BuildEventOutput(ctx context.Context, bazelrc string, args ...string) iter.Seq2[BuildEventOutput, error] {
	return func(yield func(BuildEventOutput, error) bool) {
		f, err := os.CreateTemp("", "snapshots-collect")
		if err != nil {
			yield(BuildEventOutput{}, fmt.Errorf("failed to create temporary file: %w", err))
			return
		}
		defer func() {
			_ = f.Close()
			_ = os.Remove(f.Name())
		}()

		args = append([]string{"build", fmt.Sprintf("--build_event_json_file=%s", f.Name())}, args...)

		if bazelrc != "" {
			args = append([]string{fmt.Sprintf("--bazelrc=%s", bazelrc)}, args...)
		}

		if _, err := c.Command(ctx, args...); err != nil {
			yield(BuildEventOutput{}, fmt.Errorf("failed to build: %w", err))
			return
		}

		for ev, err := range ParseBuildEventsFile(f) {
			if !yield(ev, err) {
				return
			}
		}
	}
}

// ParseBuildEventsFile returns an iterator over the build events
// in the given reader.
//
// The iterator is not re-usable.
func ParseBuildEventsFile(r io.Reader) iter.Seq2[BuildEventOutput, error] {
	return func(yield func(BuildEventOutput, error) bool) {
		dec := json.NewDecoder(r)
		for {
			var beo BuildEventOutput
			if err := dec.Decode(&beo); err != nil {
				if errors.Is(err, io.EOF) {
					return
				}

				yield(BuildEventOutput{}, fmt.Errorf("error parsing build event file: %w", err))
				return

			}

			if !yield(beo, nil) {
				return
			}
		}
	}
}
