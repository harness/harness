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

package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var (
	reGitHeaderPerson = regexp.MustCompile(`^(.+) <(.+)?> (\d+) (([-|+]\d\d)(\d\d))$`)
)

func ObjectHeaderIdentity(line string) (name string, email string, timestamp time.Time, err error) {
	data := reGitHeaderPerson.FindStringSubmatch(line)
	if data == nil {
		// Returns empty entityType if the header line doesn't match the pattern.
		return "", "", time.Time{}, nil
	}

	tzName := data[4]

	unixTime, err := strconv.ParseInt(data[3], 10, 64)
	if err != nil {
		return "", "", time.Time{},
			fmt.Errorf("failed to parse unix time in %q: %w", line, err)
	}

	hour, err := strconv.Atoi(data[5])
	if err != nil {
		return "", "", time.Time{},
			fmt.Errorf("unrecognized hour tz offset in %q: %w", line, err)
	}

	minutes, err := strconv.Atoi(data[6])
	if err != nil {
		return "", "", time.Time{},
			fmt.Errorf("unrecognized minute tz offset in %q: %w", line, err)
	}

	offset := hour * 60
	if hour < 0 {
		offset -= minutes
	} else {
		offset += minutes
	}
	offset *= 60

	name = data[1]
	email = data[2]
	timestamp = time.Unix(unixTime, 0).In(time.FixedZone(tzName, offset))

	return name, email, timestamp, nil
}
