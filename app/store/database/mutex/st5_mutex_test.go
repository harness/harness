// Copyright 2026 Harness, Inc.
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

package mutex_test

import (
	"sync"
	"testing"

	"github.com/harness/gitness/app/store/database/mutex"

	"github.com/stretchr/testify/require"
)

func TestSt5GlobalMutex(t *testing.T) {
	const writers = 8
	counter := 0

	var wg sync.WaitGroup
	for range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mutex.Lock()
			defer mutex.Unlock()
			counter++
		}()
	}
	wg.Wait()
	require.Equal(t, writers, counter)

	// multiple readers may hold the lock at the same time.
	mutex.RLock()
	mutex.RLock()
	require.Equal(t, writers, counter)
	mutex.RUnlock()
	mutex.RUnlock()

	// a writer waits for the readers to leave.
	mutex.RLock()
	acquired := make(chan struct{})
	go func() {
		mutex.Lock()
		close(acquired)
		mutex.Unlock()
	}()
	select {
	case <-acquired:
		t.Fatal("writer acquired the lock while a reader holds it")
	default:
	}
	mutex.RUnlock()
	<-acquired
}
