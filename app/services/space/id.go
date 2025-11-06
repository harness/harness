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

package space

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/harness/gitness/job"
)

const jobIDPrefix = "space-move-"

func (s *Service) getJobDef(jobUID string, input Input) (job.Definition, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return job.Definition{}, fmt.Errorf("failed to marshal job input json: %w", err)
	}

	strData := strings.TrimSpace(string(data))

	encryptedData, err := s.encrypter.Encrypt(strData)
	if err != nil {
		return job.Definition{}, fmt.Errorf("failed to encrypt job input: %w", err)
	}

	return job.Definition{
		UID:        jobUID,
		Type:       jobType,
		MaxRetries: moveJobMaxRetries,
		Timeout:    moveJobMaxDuration,
		Data:       base64.StdEncoding.EncodeToString(encryptedData),
	}, nil
}

func (s *Service) getJobInput(data string) (Input, error) {
	encrypted, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return Input{}, fmt.Errorf("failed to base64 decode job input: %w", err)
	}

	decrypted, err := s.encrypter.Decrypt(encrypted)
	if err != nil {
		return Input{}, fmt.Errorf("failed to decrypt job input: %w", err)
	}

	var input Input

	err = json.NewDecoder(strings.NewReader(decrypted)).Decode(&input)
	if err != nil {
		return Input{}, fmt.Errorf("failed to unmarshal job input json: %w", err)
	}

	return input, nil
}

func (s *Service) JobIDFromSpaceIdentifier(srcSpaceIdentifier string) string {
	return jobIDPrefix + srcSpaceIdentifier
}
