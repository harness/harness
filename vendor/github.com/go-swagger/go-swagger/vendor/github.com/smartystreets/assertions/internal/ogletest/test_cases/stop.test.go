// Copyright 2015 Aaron Jacobs. All Rights Reserved.
// Author: aaronjjacobs@gmail.com (Aaron Jacobs)
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

package oglematchers_test

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/assertions/internal/ogletest"
)

func TestStop(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type StopTest struct {
}

var _ TearDownInterface = &StopTest{}
var _ TearDownTestSuiteInterface = &StopTest{}

func init() { RegisterTestSuite(&StopTest{}) }

func (t *StopTest) TearDown() {
	fmt.Println("TearDown running.")
}

func (t *StopTest) TearDownTestSuite() {
	fmt.Println("TearDownTestSuite running.")
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *StopTest) First() {
}

func (t *StopTest) Second() {
	fmt.Println("About to call StopRunningTests.")
	StopRunningTests()
	fmt.Println("Called StopRunningTests.")
}

func (t *StopTest) Third() {
}
