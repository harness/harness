package goblin

import (
	"fmt"
	"reflect"
	"strings"
)

type Assertion struct {
	src  interface{}
	fail func(interface{})
}

func objectsAreEqual(a, b interface{}) bool {
	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		return false
	}

	if reflect.DeepEqual(a, b) {
		return true
	}

	if fmt.Sprintf("%#v", a) == fmt.Sprintf("%#v", b) {
		return true
	}

	return false
}

func formatMessages(messages ...string) string {
	if len(messages) > 0 {
		return ", " + strings.Join(messages, " ")
	}
	return ""
}

func (a *Assertion) Eql(dst interface{}) {
	a.Equal(dst)
}

func (a *Assertion) Equal(dst interface{}) {
	if !objectsAreEqual(a.src, dst) {
		a.fail(fmt.Sprintf("%#v %s %#v", a.src, "does not equal", dst))
	}
}

func (a *Assertion) IsTrue(messages ...string) {
	if !objectsAreEqual(a.src, true) {
		message := fmt.Sprintf("%v %s%s", a.src, "expected false to be truthy", formatMessages(messages...))
		a.fail(message)
	}
}

func (a *Assertion) IsFalse(messages ...string) {
	if !objectsAreEqual(a.src, false) {
		message := fmt.Sprintf("%v %s%s", a.src, "expected true to be falsey", formatMessages(messages...))
		a.fail(message)
	}
}
