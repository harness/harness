//  Copyright 2023 Harness, Inc.
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

package utils

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

func MatchArtifactFilter(
	allowedPattern pq.StringArray,
	blockedPattern pq.StringArray, artifact string,
) (bool, error) {
	allowedPatterns := []string(allowedPattern)
	blockedPatterns := []string(blockedPattern)

	if len(blockedPatterns) > 0 {
		flag, err := matchPatterns(blockedPatterns, artifact)
		if err != nil {
			return flag, fmt.Errorf(
				"failed to match blocked patterns for artifact %s: %w",
				artifact, err,
			)
		}
		if flag {
			return false, errors.New(
				"failed because artifact seems to be matching blocked patterns configured on repository",
			)
		}
	}

	if len(allowedPatterns) > 0 {
		flag, err := matchPatterns(allowedPatterns, artifact)
		if err != nil {
			return flag, fmt.Errorf(
				"failed to match allowed patterns for artifact %s: %w",
				artifact, err,
			)
		}

		if !flag {
			return false, errors.New(
				"failed because artifact doesn't seems to be matching allowed patterns configured on repository",
			)
		}
	}
	return true, nil
}

func matchPatterns(
	patterns []string,
	val string,
) (bool, error) {
	for _, pattern := range patterns {
		flag, err := regexp.MatchString(pattern, val)
		if err != nil {
			log.Error().Err(err).Msgf(
				"failed to match pattern %s for val %s",
				pattern,
				val,
			)
			return flag, fmt.Errorf(
				"failed to match pattern %s for val %s: %w",
				pattern,
				val,
				err,
			)
		}
		if flag {
			return true, nil
		}
	}
	return false, nil
}
