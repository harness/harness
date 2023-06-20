// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
