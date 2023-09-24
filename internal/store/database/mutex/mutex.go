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

// Package mutex provides a global mutex.
package mutex

import "sync"

var m sync.RWMutex

// RLock locks the global mutex for reads.
func RLock() { m.RLock() }

// RUnlock unlocks the global mutex.
func RUnlock() { m.RUnlock() }

// Lock locks the global mutex for writes.
func Lock() { m.Lock() }

// Unlock unlocks the global mutex.
func Unlock() { m.Unlock() }
