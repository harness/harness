package reporting

import (
	"runtime"
	"testing"
)

func TestEachNestedReporterReceivesTheCallFromTheContainingReporter(t *testing.T) {
	fake1 := newFakeReporter()
	fake2 := newFakeReporter()
	reporter := NewReporters(fake1, fake2)

	reporter.BeginStory(nil)
	assertTrue(t, fake1.begun)
	assertTrue(t, fake2.begun)

	reporter.Enter(NewScopeReport("scope", "hi"))
	assertTrue(t, fake1.entered)
	assertTrue(t, fake2.entered)

	reporter.Report(NewSuccessReport())
	assertTrue(t, fake1.reported)
	assertTrue(t, fake2.reported)

	reporter.Exit()
	assertTrue(t, fake1.exited)
	assertTrue(t, fake2.exited)

	reporter.EndStory()
	assertTrue(t, fake1.ended)
	assertTrue(t, fake2.ended)
}

func assertTrue(t *testing.T, value bool) {
	if !value {
		_, _, line, _ := runtime.Caller(1)
		t.Errorf("Value should have been true (but was false). See line %d", line)
	}
}

type fakeReporter struct {
	begun    bool
	entered  bool
	reported bool
	exited   bool
	ended    bool
}

func newFakeReporter() *fakeReporter {
	return &fakeReporter{}
}

func (self *fakeReporter) BeginStory(story *StoryReport) {
	self.begun = true
}
func (self *fakeReporter) Enter(scope *ScopeReport) {
	self.entered = true
}
func (self *fakeReporter) Report(report *AssertionResult) {
	self.reported = true
}
func (self *fakeReporter) Exit() {
	self.exited = true
}
func (self *fakeReporter) EndStory() {
	self.ended = true
}
