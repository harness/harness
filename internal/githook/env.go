// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package githook

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
)

const (
	// envPayload defines the environment variable name used to send the payload to githook binary.
	// NOTE: Since the variable is not meant for gitness itself, don't prefix with 'GITNESS'.
	envPayload = "GIT_HOOK_GITNESS_PAYLOAD"
)

var (
	ErrHookDisabled = errors.New("hook disabled")
)

// Payload defines the Payload the githook binary is initiated with when executing the git hooks.
type Payload struct {
	APIBaseURL  string
	RepoID      int64
	PrincipalID int64
	RequestID   string
	Disabled    bool // this will stop processing server hooks
}

// GenerateEnvironmentVariables generates the environment variables that are sent to the githook binary.
// NOTE: for now we use a single environment variable to reduce the overal number of environment variables.
func GenerateEnvironmentVariables(payload *Payload) (map[string]string, error) {
	// serialize the payload
	payloadBuff := &bytes.Buffer{}
	encoder := gob.NewEncoder(payloadBuff)
	if err := encoder.Encode(payload); err != nil {
		return nil, fmt.Errorf("failed to encode payload: %w", err)
	}

	// send it as base64 to avoid issues with any problematic characters
	// NOTE: this will blow up the payload by ~33%, though it's not expected to be too big.
	// On the other hand, we save a lot of size by only needing one environment variable name.
	payloadBase64 := base64.StdEncoding.EncodeToString(payloadBuff.Bytes())

	return map[string]string{
		envPayload: payloadBase64,
	}, nil
}

// loadPayloadFromEnvironment loads the githook payload from the environment.
func loadPayloadFromEnvironment() (*Payload, error) {
	// retrieve payload from environment variables
	payloadBase64, err := getEnvironmentVariable(envPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to load payload from environment variables: %w", err)
	}

	// decode base64
	payloadBytes, err := base64.StdEncoding.DecodeString(payloadBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to base64 decode payload: %w", err)
	}

	// deserialize the payload
	var payload Payload
	decoder := gob.NewDecoder(bytes.NewReader(payloadBytes))
	err = decoder.Decode(&payload)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize payload: %w", err)
	}

	// ensure payload is valid
	err = validatePayload(&payload)
	if err != nil {
		return nil, fmt.Errorf("payload contains invalid data: %w", err)
	}

	return &payload, nil
}

// validatePayload performs a BASIC validation of the payload.
func validatePayload(payload *Payload) error {
	if payload == nil {
		return errors.New("payload is empty")
	}
	if payload.Disabled {
		return ErrHookDisabled
	}
	if payload.APIBaseURL == "" {
		return errors.New("payload doesn't contain a base url")
	}
	if payload.PrincipalID <= 0 {
		return errors.New("payload doesn't contain a principal id")
	}
	if payload.RepoID <= 0 {
		return errors.New("payload doesn't contain a repo id")
	}

	return nil
}

func getEnvironmentVariable(name string) (string, error) {
	val, ok := os.LookupEnv(name)
	if !ok {
		return "", fmt.Errorf("'%s' not found in env", name)
	}

	if val == "" {
		return "", fmt.Errorf("'%s' found in env but it's empty", name)
	}

	return val, nil
}
