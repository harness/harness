// Oh the stack trace scanning!
// The density of comments in this file is evidence that
// the code doesn't exactly explain itself. Tread with care...
package convey

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// suiteContext magically handles all coordination of reporter, runners as they handle calls
// to Convey, So, and the like. It does this via runtime call stack inspection, making sure
// that each test function has its own runner, and routes all live registrations
// to the appropriate runner.
type suiteContext struct {
	lock    sync.Mutex
	runners map[string]*runner // key: testName;

	// stores a correlation to the actual runner for outside-of-stack scenaios
	locations map[string]string // key: file:line; value: testName (key to runners)
}

func (self *suiteContext) Run(entry *registration) {
	if self.current() != nil {
		panic(extraGoTest)
	}

	reporter := buildReporter()
	runner := newRunner()
	runner.UpgradeReporter(reporter)

	testName, location, _ := suiteAnchor()

	self.lock.Lock()
	self.locations[location] = testName
	self.runners[testName] = runner
	self.lock.Unlock()

	runner.Begin(entry)
	runner.Run()

	self.lock.Lock()
	delete(self.locations, location)
	delete(self.runners, testName)
	self.lock.Unlock()
}

func (self *suiteContext) Current() *runner {
	if runner := self.current(); runner != nil {
		return runner
	}
	panic(missingGoTest)
}
func (self *suiteContext) current() *runner {
	self.lock.Lock()
	defer self.lock.Unlock()
	testName, _, err := suiteAnchor()

	if err != nil {
		testName = correlate(self.locations)
	}

	return self.runners[testName]
}

func newSuiteContext() *suiteContext {
	self := new(suiteContext)
	self.locations = make(map[string]string)
	self.runners = make(map[string]*runner)
	return self
}

//////////////////// Helper Functions ///////////////////////

// suiteAnchor returns the enclosing test function name (including package) and the
// file:line combination of the top-level Convey. It does this by traversing the
// call stack in reverse, looking for the go testing harnass call ("testing.tRunner")
// and then grabs the very next entry.
func suiteAnchor() (testName, location string, err error) {
	callers := runtime.Callers(0, callStack)

	for y := callers; y > 0; y-- {
		callerId, file, conveyLine, found := runtime.Caller(y)
		if !found {
			continue
		}

		if name := runtime.FuncForPC(callerId).Name(); name != goTestHarness {
			continue
		}

		callerId, file, conveyLine, _ = runtime.Caller(y - 1)
		testName = runtime.FuncForPC(callerId).Name()
		location = fmt.Sprintf("%s:%d", file, conveyLine)
		return
	}
	return "", "", errors.New("Can't resolve test method name! Are you calling Convey() from a `*_test.go` file and a `Test*` method (because you should be)?")
}

// correlate links the current stack with the appropriate
// top-level Convey by comparing line numbers in its own stack trace
// with the registered file:line combo. It's come to this.
func correlate(locations map[string]string) (testName string) {
	file, line := resolveTestFileAndLine()
	closest := -1
	for location, registeredTestName := range locations {
		parts := strings.Split(location, ":")
		locationFile := parts[0]
		if locationFile != file {
			continue
		}

		locationLine, err := strconv.Atoi(parts[1])
		if err != nil || locationLine < line {
			continue
		}

		if closest == -1 || locationLine < closest {
			closest = locationLine
			testName = registeredTestName
		}
	}
	return
}

// resolveTestFileAndLine is used as a last-ditch effort to correlate an
// assertion with the right executor and runner.
func resolveTestFileAndLine() (file string, line int) {
	callers := runtime.Callers(0, callStack)
	var found bool

	for y := callers; y > 0; y-- {
		_, file, line, found = runtime.Caller(y)
		if !found {
			continue
		}

		if strings.HasSuffix(file, "_test.go") {
			return
		}
	}
	return "", 0
}

const maxStackDepth = 100               // This had better be enough...
const goTestHarness = "testing.tRunner" // I hope this doesn't change...

var callStack []uintptr = make([]uintptr, maxStackDepth, maxStackDepth)
