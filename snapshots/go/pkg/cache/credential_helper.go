/* Copyright 2022 Cognite AS */

package cache

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type CredentialHelper struct {
	Cmd *exec.Cmd
}

type Credentials struct {
	Headers struct {
		Authorization []string `json:"Authorization"`
	} `json:"headers"`
}

func NewCredentialHelper(cmd *exec.Cmd) *CredentialHelper {
	return &CredentialHelper{Cmd: cmd}
}

func (ch *CredentialHelper) GetAuthorization() ([]string, error) {
	// Having no credential helper isn't an error
	if ch.Cmd == nil {
		return nil, nil
	}

	headers, err := ch.Cmd.Output()
	if err != nil {
		return nil, err
	}

	credentials := &Credentials{}
	if err := json.Unmarshal(headers, &credentials); err != nil {
		return nil, fmt.Errorf("invalid headers %s: %w", headers, err)
	}

	return credentials.Headers.Authorization, nil
}
