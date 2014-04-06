package convey

import (
	"reflect"
	"runtime"

	"github.com/smartystreets/goconvey/convey/gotest"
)

type registration struct {
	Situation string
	action    *action
	Test      t
	File      string
	Line      int
	Focus     bool
}

func (self *registration) IsTopLevel() bool {
	return self.Test != nil
}

func newRegistration(situation string, action *action, test t) *registration {
	file, line, _ := gotest.ResolveExternalCaller()
	self := new(registration)
	self.Situation = situation
	self.action = action
	self.Test = test
	self.File = file
	self.Line = line
	return self
}

////////////////////////// action ///////////////////////

type action struct {
	wrapped func()
	name    string
}

func (self *action) Invoke() {
	self.wrapped()
}

func newAction(wrapped func()) *action {
	self := new(action)
	self.name = functionName(wrapped)
	self.wrapped = wrapped
	return self
}

func newSkippedAction(wrapped func()) *action {
	self := new(action)

	// The choice to use the filename and line number as the action name
	// reflects the need for something unique but also that corresponds
	// in a determinist way to the action itself.
	self.name = gotest.FormatExternalFileAndLine()
	self.wrapped = wrapped
	return self
}

///////////////////////// helpers //////////////////////////////

func functionName(action func()) string {
	return runtime.FuncForPC(functionId(action)).Name()
}

func functionId(action func()) uintptr {
	return reflect.ValueOf(action).Pointer()
}
