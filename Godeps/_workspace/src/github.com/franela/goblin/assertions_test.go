package goblin

import "testing"

// So we can test asserting type against its type alias
type String string

// Helper for testing Assertion conditions
type AssertionVerifier struct {
	ShouldPass bool
	didFail    bool
	msg        interface{}
}

func (a *AssertionVerifier) FailFunc(msg interface{}) {
	a.didFail = true
	a.msg = msg
}

func (a *AssertionVerifier) Verify(t *testing.T) {
	if a.didFail == a.ShouldPass {
		t.FailNow()
	}
}

func (a *AssertionVerifier) VerifyMessage(t *testing.T, message string) {
	a.Verify(t)
	if a.msg.(string) != message {
		t.Fatalf(`"%s" != "%s"`, a.msg, message)
	}

}

func TestEqual(t *testing.T) {

	verifier := AssertionVerifier{ShouldPass: true}
	a := Assertion{src: 1, fail: verifier.FailFunc}
	a.Equal(1)
	verifier.Verify(t)
	a.Eql(1)
	verifier.Verify(t)

	a = Assertion{src: "foo"}
	a.Equal("foo")
	verifier.Verify(t)
	a.Eql("foo")
	verifier.Verify(t)

	a = Assertion{src: map[string]string{"foo": "bar"}}
	a.Equal(map[string]string{"foo": "bar"})
	verifier.Verify(t)
	a.Eql(map[string]string{"foo": "bar"})
	verifier.Verify(t)

	verifier = AssertionVerifier{ShouldPass: false}
	a = Assertion{src: String("baz"), fail: verifier.FailFunc}
	a.Equal("baz")
	verifier.Verify(t)
	a.Eql("baz")
	verifier.Verify(t)
}

func TestIsTrue(t *testing.T) {
	verifier := AssertionVerifier{ShouldPass: true}
	a := Assertion{src: true, fail: verifier.FailFunc}
	a.IsTrue()
	verifier.Verify(t)

	verifier = AssertionVerifier{ShouldPass: false}
	a = Assertion{src: false, fail: verifier.FailFunc}
	a.IsTrue()
	verifier.Verify(t)
}

func TestIsFalse(t *testing.T) {
	verifier := AssertionVerifier{ShouldPass: true}
	a := Assertion{src: false, fail: verifier.FailFunc}
	a.IsFalse()
	verifier.Verify(t)

	verifier = AssertionVerifier{ShouldPass: false}
	a = Assertion{src: true, fail: verifier.FailFunc}
	a.IsFalse()
	verifier.Verify(t)
}

func TestIsFalseWithMessage(t *testing.T) {
	verifier := AssertionVerifier{ShouldPass: false}
	a := Assertion{src: true, fail: verifier.FailFunc}
	a.IsFalse("false is not true")
	verifier.Verify(t)
	verifier.VerifyMessage(t, "true expected true to be falsey, false is not true")
}

func TestIsTrueWithMessage(t *testing.T) {
	verifier := AssertionVerifier{ShouldPass: false}
	a := Assertion{src: false, fail: verifier.FailFunc}
	a.IsTrue("true is not false")
	verifier.Verify(t)
	verifier.VerifyMessage(t, "false expected false to be truthy, true is not false")
}
