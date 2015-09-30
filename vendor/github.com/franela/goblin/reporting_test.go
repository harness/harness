package goblin

import (
	"reflect"
	"testing"
	"time"
)

type FakeReporter struct {
	describes          []string
	fails              []string
	passes             []string
	pending            []string
	ends               int
	failures           int
	executionTime      time.Duration
	totalExecutionTime time.Duration
	beginFlag, endFlag bool
}

func (r *FakeReporter) beginDescribe(name string) {
	r.describes = append(r.describes, name)
}

func (r *FakeReporter) endDescribe() {
	r.ends++
}

func (r *FakeReporter) failure(failure *Failure) {
	r.failures++
}

func (r *FakeReporter) itFailed(name string) {
	r.fails = append(r.fails, name)
}

func (r *FakeReporter) itPassed(name string) {
	r.passes = append(r.passes, name)
}

func (r *FakeReporter) itIsPending(name string) {
	r.pending = append(r.pending, name)
}

func (r *FakeReporter) itTook(duration time.Duration) {
	r.executionTime = duration
	r.totalExecutionTime += duration
}

func (r *FakeReporter) begin() {
	r.beginFlag = true
}

func (r *FakeReporter) end() {
	r.endFlag = true
}

func TestReporting(t *testing.T) {
	fakeTest := &testing.T{}
	reporter := FakeReporter{}
	fakeReporter := Reporter(&reporter)

	g := Goblin(fakeTest)
	g.SetReporter(fakeReporter)

	g.Describe("One", func() {
		g.It("Foo", func() {
			g.Assert(0).Equal(1)
		})
		g.Describe("Two", func() {
			g.It("Bar", func() {
				g.Assert(0).Equal(0)
			})
		})
	})

	if !reflect.DeepEqual(reporter.describes, []string{"One", "Two"}) {
		t.FailNow()
	}
	if !reflect.DeepEqual(reporter.fails, []string{"Foo"}) {
		t.FailNow()
	}
	if !reflect.DeepEqual(reporter.passes, []string{"Bar"}) {
		t.FailNow()
	}
	if reporter.ends != 2 {
		t.FailNow()
	}

	if !reporter.beginFlag || !reporter.endFlag {
		t.FailNow()
	}
}

func TestReportingTime(t *testing.T) {
	fakeTest := &testing.T{}
	reporter := FakeReporter{}
	fakeReporter := Reporter(&reporter)

	g := Goblin(fakeTest)
	g.SetReporter(fakeReporter)

	g.Describe("One", func() {
		g.AfterEach(func() {
			//TODO: Make this an assertion
			if int64(reporter.executionTime/time.Millisecond) < 5 || int64(reporter.executionTime/time.Millisecond) >= 6 {
				t.FailNow()
			}
		})
		g.It("Foo", func() {
			time.Sleep(5 * time.Millisecond)
		})
		g.Describe("Two", func() {
			g.It("Bar", func() {
				time.Sleep(5 * time.Millisecond)
			})
		})
	})

	if int64(reporter.totalExecutionTime/time.Millisecond) < 10 {
		t.FailNow()
	}
}

func TestReportingPending(t *testing.T) {
	fakeTest := &testing.T{}
	reporter := FakeReporter{}
	fakeReporter := Reporter(&reporter)

	g := Goblin(fakeTest)
	g.SetReporter(fakeReporter)

	g.Describe("One", func() {
		g.It("One")
		g.Describe("Two", func() {
			g.It("Two")
		})
	})

	if !reflect.DeepEqual(reporter.pending, []string{"One", "Two"}) {
		t.FailNow()
	}
}

func TestReportingErrors(t *testing.T) {
	fakeTest := &testing.T{}
	reporter := FakeReporter{}
	fakeReporter := Reporter(&reporter)

	g := Goblin(fakeTest)
	g.SetReporter(fakeReporter)

	g.Describe("Numbers", func() {
		g.It("Should make reporting add two errors ", func() {
			g.Assert(0).Equal(1)
			g.Assert(0).Equal(1)
		})
	})

	if reporter.failures != 1 {
		t.FailNow()
	}
}
