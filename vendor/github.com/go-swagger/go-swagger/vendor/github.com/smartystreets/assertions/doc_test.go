package assertions

import (
	"bytes"
	"fmt"
	"testing"
)

func TestPassingAssertion(t *testing.T) {
	fake := &FakeT{buffer: new(bytes.Buffer)}
	assertion := New(fake)
	passed := assertion.So(1, ShouldEqual, 1)

	if !passed {
		t.Error("Assertion failed when it should have passed.")
	}
	if fake.buffer.Len() > 0 {
		t.Error("Unexpected error message was printed.")
	}
}

func TestFailingAssertion(t *testing.T) {
	fake := &FakeT{buffer: new(bytes.Buffer)}
	assertion := New(fake)
	passed := assertion.So(1, ShouldEqual, 2)

	if passed {
		t.Error("Assertion passed when it should have failed.")
	}
	if fake.buffer.Len() == 0 {
		t.Error("Expected error message not printed.")
	}
}

func TestFailingGroupsOfAssertions(t *testing.T) {
	fake := &FakeT{buffer: new(bytes.Buffer)}
	assertion1 := New(fake)
	assertion2 := New(fake)

	assertion1.So(1, ShouldEqual, 2) // fail
	assertion2.So(1, ShouldEqual, 1) // pass

	if !assertion1.Failed() {
		t.Error("Expected the first assertion to have been marked as failed.")
	}
	if assertion2.Failed() {
		t.Error("Expected the second assertion to NOT have been marked as failed.")
	}
}

type FakeT struct {
	buffer *bytes.Buffer
}

func (this *FakeT) Error(args ...interface{}) {
	fmt.Fprint(this.buffer, args...)
}
