/* Copyright 2022 Cognite AS */

// package bazel provides a simple Bazel API for the specific functionality
// needed in snapshots.
package bazel

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

	fmt.Printf("args: %v", args)

	cmd := exec.CommandContext(ctx, c.path, args...)
	cmd.Stderr = c.stderr
	cmd.Stdout = buf
	cmd.Dir = c.ws

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("bazel command error: %w", err)
	}

	return buf.Bytes(), nil
}

func (c *Client) BuildEventOutput(ctx context.Context, bazelrc string, args ...string) ([]BuildEventOutput, error) {
	// create a temporary file
	f, err := os.CreateTemp("", "snapshots-collect")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(f.Name())

	args = append([]string{"build", fmt.Sprintf("--build_event_json_file=%s", f.Name())}, args...)

	if bazelrc != "" {
		args = append([]string {fmt.Sprintf("--bazelrc=%s", bazelrc)}, args...)
	}

	if _, err := c.Command(ctx, args...); err != nil {
		return nil, fmt.Errorf("failed to build: %w", err)
	}

	buildEvents := make([]BuildEventOutput, 0)
	reader := bufio.NewReader(f)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error reading build event file: %w", err)
		}

		beo := BuildEventOutput{}
		if err := json.Unmarshal(line, &beo); err != nil {
			return nil, fmt.Errorf("error parsing build event file: %w", err)
		}

		buildEvents = append(buildEvents, beo)
	}

	return buildEvents, nil
}
