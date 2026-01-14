// Source: https://github.com/distribution/distribution

// Copyright 2014 https://github.com/distribution/distribution Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dcontext

import (
	"runtime"
	"testing"
	"time"
)

// TestWithTrace ensures that tracing has the expected values in the context.
func TestWithTrace(t *testing.T) {
	t.Parallel()
	pc, file, _, _ := runtime.Caller(0) // get current caller.
	f := runtime.FuncForPC(pc)

	base := []valueTestCase{
		{
			key:           "trace.id",
			notnilorempty: true,
		},

		{
			key:           "trace.file",
			expected:      file,
			notnilorempty: true,
		},
		{
			key:           "trace.line",
			notnilorempty: true,
		},
		{
			key:           "trace.start",
			notnilorempty: true,
		},
	}

	ctx, done := WithTrace(Background())
	t.Cleanup(func() { done("this will be emitted at end of test") })

	tests := base
	tests = append(
		tests, valueTestCase{
			key:      "trace.func",
			expected: f.Name(),
		},
	)
	for _, tc := range tests {
		testCase := tc
		t.Run(
			testCase.key, func(t *testing.T) {
				t.Parallel()
				v := ctx.Value(testCase.key)
				if testCase.notnilorempty {
					if v == nil || v == "" {
						t.Fatalf("value was nil or empty: %#v", v)
					}
					return
				}

				if v != testCase.expected {
					t.Fatalf("unexpected value: %v != %v", v, testCase.expected)
				}
			},
		)
	}

	tracedFn := func() {
		parentID := ctx.Value("trace.id") // ensure the parent trace id is correct.

		pc1, _, _, _ := runtime.Caller(0) // get current caller.
		f1 := runtime.FuncForPC(pc1)
		ctx, done1 := WithTrace(ctx)
		defer done1("this should be subordinate to the other trace")
		time.Sleep(time.Second)
		tests1 := base
		tests1 = append(
			tests1, valueTestCase{
				key:      "trace.func",
				expected: f1.Name(),
			}, valueTestCase{
				key:      "trace.parent.id",
				expected: parentID,
			},
		)
		for _, tc := range tests1 {
			testCase := tc
			t.Run(
				testCase.key, func(t *testing.T) {
					t.Parallel()
					v := ctx.Value(testCase.key)
					if testCase.notnilorempty {
						if v == nil || v == "" {
							t.Fatalf("value was nil or empty: %#v", v)
						}
						return
					}

					if v != testCase.expected {
						t.Fatalf("unexpected value: %v != %v", v, testCase.expected)
					}
				},
			)
		}
	}
	tracedFn()

	time.Sleep(time.Second)
}

type valueTestCase struct {
	key           string
	expected      interface{}
	notnilorempty bool // just check not empty/not nil
}
