/* Copyright 2022 Cognite AS */

package collecter

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type Credentials struct {
	Headers struct {
		Authorization []string `json:"Authorization"`
	} `json:"headers"`
}

func getAuthorization(credentialHelper, workspacePath string) ([]string, error) {
	if credentialHelper == "" {
		return nil, nil
	}

	cmd := exec.Cmd{Path: credentialHelper, Dir: workspacePath}

	headers, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	credentials := &Credentials{}
	if err := json.Unmarshal(headers, &credentials); err != nil {
		return nil, fmt.Errorf("invalid headers %s: %w", headers, err)
	}

	if len(credentials.Headers.Authorization) == 0 {
		return nil, fmt.Errorf("empty authorization header")
	}

	return credentials.Headers.Authorization, nil
}
