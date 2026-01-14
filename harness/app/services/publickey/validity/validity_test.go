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

package validity

import (
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

func TestPeriodRevoke(t *testing.T) {
	reasonRetired := packet.KeyRetired
	reasonCompromised := packet.KeyCompromised
	date0 := time.Date(2010, time.February, 14, 0, 0, 0, 0, time.UTC)
	date1 := time.Date(2010, time.February, 19, 0, 0, 0, 0, time.UTC)
	dateNeg := time.Date(2010, time.January, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		validity    Period
		revocations []*packet.Signature
		expected    Period
	}{
		{
			name: "no-expiration",
			validity: Period{
				CreatedAt: date0,
			},
			revocations: []*packet.Signature{
				{
					CreationTime:     date1,
					RevocationReason: &reasonRetired,
				},
			},
			expected: Period{
				CreatedAt: date0,
				Duration:  date1.Sub(date0),
			},
		},
		{
			name: "7-day-expiration",
			validity: Period{
				CreatedAt: date0,
				Duration:  7 * 24 * time.Hour,
			},
			revocations: []*packet.Signature{
				{
					CreationTime:     date1,
					RevocationReason: &reasonRetired,
				},
			},
			expected: Period{
				CreatedAt: date0,
				Duration:  date1.Sub(date0),
			},
		},
		{
			name: "1-day-expiration",
			validity: Period{
				CreatedAt: date0,
				Duration:  24 * time.Hour,
			},
			revocations: []*packet.Signature{
				{
					CreationTime:     date1,
					RevocationReason: &reasonRetired,
				},
			},
			expected: Period{
				CreatedAt: date0,
				Duration:  24 * time.Hour,
			},
		},
		{
			name: "revocation-in-past",
			validity: Period{
				CreatedAt: date0,
				Duration:  24 * time.Hour,
			},
			revocations: []*packet.Signature{
				{
					CreationTime:     dateNeg,
					RevocationReason: &reasonRetired,
				},
			},
			expected: Period{
				Invalid:   true,
				CreatedAt: date0,
			},
		},
		{
			name: "compromised",
			validity: Period{
				CreatedAt: date0,
			},
			revocations: []*packet.Signature{
				{
					CreationTime:     date1,
					RevocationReason: &reasonCompromised,
				},
			},
			expected: Period{
				Invalid:   true,
				CreatedAt: date0,
			},
		},
		{
			name: "compromised",
			validity: Period{
				Invalid: true,
			},
			revocations: []*packet.Signature{
				{
					CreationTime:     date1,
					RevocationReason: &reasonRetired,
				},
			},
			expected: Period{
				Invalid: true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.validity.Revoke(test.revocations)
			if want, got := test.expected, test.validity; want != got {
				t.Errorf("failed: want=%v got=%v", want, got)
			}
		})
	}
}

func TestPeriodIntersect(t *testing.T) {
	date0 := time.Date(2010, time.October, 14, 0, 0, 0, 0, time.UTC)
	date1 := time.Date(2010, time.October, 19, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2010, time.October, 21, 0, 0, 0, 0, time.UTC)
	date3 := time.Date(2010, time.October, 28, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		validity  Period
		validity2 Period
		expected  Period
	}{
		{
			name:      "no-expiration-v2-after-v1",
			validity:  Period{CreatedAt: date0}, // v1:  1111111
			validity2: Period{CreatedAt: date1}, // v2:  ...2222
			expected:  Period{CreatedAt: date1}, // res: ...3333
		},
		{
			name:      "no-expiration-v1-after-v2",
			validity:  Period{CreatedAt: date1}, // v1:  ...1111
			validity2: Period{CreatedAt: date0}, // v2:  2222222
			expected:  Period{CreatedAt: date1}, // res: ...3333
		},
		{
			name:      "v2-overlaps-at-the-end",
			validity:  fromTimes(date0, date2), // v1:  11111..
			validity2: fromTimes(date1, date3), // v2:  ...2222
			expected:  fromTimes(date1, date2), // res: ...33..
		},
		{
			name:      "v2-overlaps-at-the-start",
			validity:  fromTimes(date1, date3), // v1:  ...1111
			validity2: fromTimes(date0, date2), // v2:  22222..
			expected:  fromTimes(date1, date2), // res: ...33..
		},
		{
			name:      "v1-no-exp;v2-after",
			validity:  Period{CreatedAt: date0}, // v1:  1111111
			validity2: fromTimes(date1, date3),  // v2:  ...2222
			expected:  fromTimes(date1, date3),  // res: ...3333
		},
		{
			name:      "v1-no-exp;v2-before",
			validity:  Period{CreatedAt: date1}, // v1:  ..11111
			validity2: fromTimes(date0, date3),  // v2:  2222222
			expected:  fromTimes(date1, date3),  // res: ..33333
		},
		{
			name:      "v2-no-exp;v2-after",
			validity:  fromTimes(date0, date3),  // v1:  1111111
			validity2: Period{CreatedAt: date1}, // v2:  ..22222
			expected:  fromTimes(date1, date3),  // res: ..33333
		},
		{
			name:      "v2-no-exp;v2-before",
			validity:  fromTimes(date1, date3),  // v1:  ..11111
			validity2: Period{CreatedAt: date0}, // v2:  2222222
			expected:  fromTimes(date1, date3),  // res: ..33333
		},
		{
			name:      "v2-is-subperiod",
			validity:  fromTimes(date0, date3), // v1:  1111111
			validity2: fromTimes(date1, date2), // v2:  ..222..
			expected:  fromTimes(date1, date2), // res: ..333..
		},
		{
			name:      "v2-is-superperiod",
			validity:  fromTimes(date1, date2), // v1:  ..111..
			validity2: fromTimes(date0, date3), // v2:  2222222
			expected:  fromTimes(date1, date2), // res: ..333..
		},
		{
			name:      "no-overlap",
			validity:  fromTimes(date0, date1), // v1:  111....
			validity2: fromTimes(date2, date3), // v2:  ....222
			expected:  Period{Invalid: true, CreatedAt: date0},
		},
		{
			name:      "v1-invalid",
			validity:  Period{Invalid: true, CreatedAt: date1},
			validity2: fromTimes(date0, date3),
			expected:  Period{Invalid: true, CreatedAt: date1},
		},
		{
			name:      "v2-invalid",
			validity:  fromTimes(date0, date3),
			validity2: Period{Invalid: true, CreatedAt: date1},
			expected:  Period{Invalid: true, CreatedAt: date0},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.validity.Intersect(test.validity2)
			if want, got := test.expected, test.validity; want != got {
				t.Errorf("failed: want=%v got=%v", want, got)
			}
		})
	}
}

func fromTimes(from, to time.Time) Period {
	dur := to.Sub(from)
	if dur <= 0 {
		return Period{Invalid: true}
	}
	return Period{
		CreatedAt: from,
		Duration:  dur,
	}
}
