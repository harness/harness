// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hook

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"os"
)

const (
	// envNamePayload defines the environment variable name used to send the payload to githook binary.
	envNamePayload = "GIT_HOOK_PAYLOAD"
)

// GenerateEnvironmentVariables generates the environment variables that should be used when calling git
// to ensure the payload will be available to the githook cli.
func GenerateEnvironmentVariables(payload any) (map[string]string, error) {
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
		envNamePayload: payloadBase64,
	}, nil
}

// LoadPayloadFromMap loads the payload from a map containing environment variables in a map format.
func LoadPayloadFromMap[T any](envVars map[string]string) (T, error) {
	var payload T

	// retrieve payload from environment variables
	payloadBase64, ok := envVars[envNamePayload]
	if !ok {
		return payload, fmt.Errorf("environment variable %q not found", envNamePayload)
	}

	return decodePayload[T](payloadBase64)
}

// LoadPayloadFromEnvironment loads the githook payload from the environment.
func LoadPayloadFromEnvironment[T any]() (T, error) {
	var payload T

	// retrieve payload from environment variables
	payloadBase64, err := getRequiredEnvironmentVariable(envNamePayload)
	if err != nil {
		return payload, fmt.Errorf("failed to load payload from environment variables: %w", err)
	}

	return decodePayload[T](payloadBase64)
}

func decodePayload[T any](encodedPayload string) (T, error) {
	var payload T
	// decode base64
	payloadBytes, err := base64.StdEncoding.DecodeString(encodedPayload)
	if err != nil {
		return payload, fmt.Errorf("failed to base64 decode payload: %w", err)
	}

	// deserialize the payload
	decoder := gob.NewDecoder(bytes.NewReader(payloadBytes))
	err = decoder.Decode(&payload)
	if err != nil {
		return payload, fmt.Errorf("failed to deserialize payload: %w", err)
	}

	return payload, nil
}

func getRequiredEnvironmentVariable(name string) (string, error) {
	val, ok := os.LookupEnv(name)
	if !ok {
		return "", fmt.Errorf("environment variable %q not found", name)
	}

	if val == "" {
		return "", fmt.Errorf("environment variable %q found but it's empty", name)
	}

	return val, nil
}
