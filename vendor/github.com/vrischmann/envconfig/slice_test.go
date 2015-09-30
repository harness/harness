package envconfig

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestSliceTokenizer(t *testing.T) {
	str := "foobar,barbaz"
	tnz := newSliceTokenizer(str)

	b := tnz.scan()
	ok(t, tnz.Err())
	equals(t, true, b)

	equals(t, "foobar", tnz.text())

	b = tnz.scan()
	ok(t, tnz.Err())
	equals(t, true, b)
	equals(t, "barbaz", tnz.text())

	b = tnz.scan()
	ok(t, tnz.Err())
	equals(t, false, b)
}

func TestSliceOfStructsTokenizer(t *testing.T) {
	str := "{foobar,100},{barbaz,200}"
	tnz := newSliceTokenizer(str)

	b := tnz.scan()
	ok(t, tnz.Err())
	equals(t, true, b)

	equals(t, "{foobar,100}", tnz.text())

	b = tnz.scan()
	ok(t, tnz.Err())
	equals(t, true, b)
	equals(t, "{barbaz,200}", tnz.text())

	b = tnz.scan()
	ok(t, tnz.Err())
	equals(t, false, b)
}

// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}
