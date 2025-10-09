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

package dbtx

import (
	"sync"

	"github.com/jmoiron/sqlx"
)

const (
	postgres = "postgres"
)

type locker interface {
	Lock()
	Unlock()
	RLock()
	RUnlock()
}

var globalMx sync.RWMutex

func needsLocking(driver string) bool {
	return driver != postgres
}

func getLocker(db *sqlx.DB) locker {
	if needsLocking(db.DriverName()) {
		return &globalMx
	}
	return lockerNop{}
}

type lockerNop struct{}

func (lockerNop) RLock()   {}
func (lockerNop) RUnlock() {}
func (lockerNop) Lock()    {}
func (lockerNop) Unlock()  {}
