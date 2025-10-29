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

package base

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRegulatorEnterExit(t *testing.T) {
	const limit = 500

	r, ok := NewRegulator(nil, limit).(*regulator)
	if !ok {
		t.Fatalf("Error: r is not of type *regulator")
		return
	}

	for range 50 {
		run := make(chan struct{})

		var firstGroupReady sync.WaitGroup
		var firstGroupDone sync.WaitGroup
		firstGroupReady.Add(limit)
		firstGroupDone.Add(limit)
		for range limit {
			go func() {
				r.enter()
				firstGroupReady.Done()
				<-run
				r.exit()
				firstGroupDone.Done()
			}()
		}
		firstGroupReady.Wait()

		// now we exhausted all the limit, let's run a little bit more
		var secondGroupReady sync.WaitGroup
		var secondGroupDone sync.WaitGroup
		for range 50 {
			secondGroupReady.Add(1)
			secondGroupDone.Add(1)
			go func() {
				secondGroupReady.Done()
				r.enter()
				r.exit()
				secondGroupDone.Done()
			}()
		}
		secondGroupReady.Wait()

		// allow the first group to return resources
		close(run)

		done := make(chan struct{})
		go func() {
			secondGroupDone.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("some r.enter() are still locked")
		}

		firstGroupDone.Wait()

		if r.available != limit {
			t.Fatalf("r.available: got %d, want %d", r.available, limit)
		}
	}
}

func TestGetLimitFromParameter(t *testing.T) {
	tests := []struct {
		Input    any
		Expected uint64
		Min      uint64
		Default  uint64
		Err      error
	}{
		{"foo", 0, 5, 5, fmt.Errorf("parameter must be an integer, 'foo' invalid")},
		{"50", 50, 5, 5, nil},
		{"5", 25, 25, 50, nil}, // lower than Min returns Min
		{nil, 50, 25, 50, nil}, // nil returns default
		{812, 812, 25, 50, nil},
	}

	for _, item := range tests {
		t.Run(
			fmt.Sprint(item.Input), func(t *testing.T) {
				actual, err := GetLimitFromParameter(item.Input, item.Min, item.Default)

				if err != nil && item.Err != nil && err.Error() != item.Err.Error() {
					t.Fatalf("GetLimitFromParameter error, expected %#v got %#v", item.Err, err)
				}

				if actual != item.Expected {
					t.Fatalf("GetLimitFromParameter result error, expected %d got %d", item.Expected, actual)
				}
			},
		)
	}
}
