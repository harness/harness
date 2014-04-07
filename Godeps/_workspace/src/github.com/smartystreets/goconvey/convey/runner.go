package convey

import (
	"fmt"

	"github.com/smartystreets/goconvey/convey/gotest"
	"github.com/smartystreets/goconvey/convey/reporting"
)

type runner struct {
	top      *scope
	chain    map[string]string
	reporter reporting.Reporter

	awaitingNewStory bool
	focus            bool
}

func (self *runner) Begin(entry *registration) {
	self.focus = entry.Focus
	self.ensureStoryCanBegin()
	self.reporter.BeginStory(reporting.NewStoryReport(entry.Test))
	self.Register(entry)
}
func (self *runner) ensureStoryCanBegin() {
	if self.awaitingNewStory {
		self.awaitingNewStory = false
	} else {
		panic(fmt.Sprintf("%s (See %s)", extraGoTest, gotest.FormatExternalFileAndLine()))
	}
}

func (self *runner) Register(entry *registration) {
	if self.focus && !entry.Focus {
		return
	}
	self.ensureStoryAlreadyStarted()
	parentAction := self.link(entry.action)
	parent := self.accessScope(parentAction)
	child := newScope(entry, self.reporter)
	parent.adopt(child)
}
func (self *runner) ensureStoryAlreadyStarted() {
	if self.awaitingNewStory {
		panic(missingGoTest)
	}
}
func (self *runner) link(action *action) string {
	_, _, parentAction := gotest.ResolveExternalCaller()
	childAction := action.name
	self.linkTo(topLevel, parentAction)
	self.linkTo(parentAction, childAction)
	return parentAction
}
func (self *runner) linkTo(value, name string) {
	if self.chain[name] == "" {
		self.chain[name] = value
	}
}
func (self *runner) accessScope(current string) *scope {
	if self.chain[current] == topLevel {
		return self.top
	}
	breadCrumbs := self.trail(current)
	return self.follow(breadCrumbs)
}
func (self *runner) trail(start string) []string {
	breadCrumbs := []string{start, self.chain[start]}
	for {
		next := self.chain[last(breadCrumbs)]
		if next == topLevel {
			break
		} else {
			breadCrumbs = append(breadCrumbs, next)
		}
	}
	return breadCrumbs[:len(breadCrumbs)-1]
}
func (self *runner) follow(trail []string) *scope {
	var accessed = self.top

	for x := len(trail) - 1; x >= 0; x-- {
		accessed = accessed.children[trail[x]]
	}
	return accessed
}

func (self *runner) RegisterReset(action *action) {
	parentAction := self.link(action)
	parent := self.accessScope(parentAction)
	parent.registerReset(action)
}

func (self *runner) Run() {
	for !self.top.visited() {
		self.top.visit()
	}
	self.reporter.EndStory()
	self.awaitingNewStory = true
}

func newRunner() *runner {
	self := new(runner)
	self.reporter = newNilReporter()
	self.top = newScope(newRegistration(topLevel, newAction(func() {}), nil), self.reporter)
	self.chain = make(map[string]string)
	self.awaitingNewStory = true
	return self
}

func (self *runner) UpgradeReporter(reporter reporting.Reporter) {
	self.reporter = reporter
}

func (self *runner) Report(result *reporting.AssertionResult) {
	self.reporter.Report(result)
	if result.Failure != "" {
		panic(failureHalt)
	}
}

func last(group []string) string {
	return group[len(group)-1]
}

const topLevel = "TOP"
const missingGoTest = `Top-level calls to Convey(...) need a reference to the *testing.T. 
    Hint: Convey("description here", t, func() { /* notice that the second argument was the *testing.T (t)! */ }) `
const extraGoTest = `Only the top-level call to Convey(...) needs a reference to the *testing.T.`
const failureHalt = "___FAILURE_HALT___"

//////////////////////// nilReporter /////////////////////////////

type nilReporter struct{}

func (self *nilReporter) BeginStory(story *reporting.StoryReport)  {}
func (self *nilReporter) Enter(scope *reporting.ScopeReport)       {}
func (self *nilReporter) Report(report *reporting.AssertionResult) {}
func (self *nilReporter) Exit()                                    {}
func (self *nilReporter) EndStory()                                {}
func newNilReporter() *nilReporter                                 { return new(nilReporter) }
