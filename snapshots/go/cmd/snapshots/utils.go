/* Copyright 2022 Cognite AS */

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func getGitHead(path string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = path

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get name from git: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}
