package goblin

import (
	"fmt"
	"reflect"
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

	if reflect.ValueOf(a) == reflect.ValueOf(b) {
		return true
	}

	if fmt.Sprintf("%#v", a) == fmt.Sprintf("%#v", b) {
		return true
	}

	return false
}

func (a *Assertion) Eql(dst interface{}) {
	a.Equal(dst)
}

func (a *Assertion) Equal(dst interface{}) {
	if !objectsAreEqual(a.src, dst) {
		a.fail(fmt.Sprintf("%v", a.src) + " does not equal " + fmt.Sprintf("%v", dst))
	}
}

func (a *Assertion) IsTrue() {
	if !objectsAreEqual(a.src, true) {
		a.fail(fmt.Sprintf("%v", a.src) + " expected false to be truthy")
	}
}

func (a *Assertion) IsFalse() {
	if !objectsAreEqual(a.src, false) {
		a.fail(fmt.Sprintf("%v", a.src) + " expected true to be falsey")
	}
}
