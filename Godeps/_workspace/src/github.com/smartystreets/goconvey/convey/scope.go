package convey

import (
	"fmt"
	"strings"

	"github.com/smartystreets/goconvey/convey/reporting"
)

type scope struct {
	name       string
	title      string
	action     *action
	children   map[string]*scope
	birthOrder []*scope
	child      int
	resets     map[string]*action
	panicked   bool
	reporter   reporting.Reporter
	report     *reporting.ScopeReport
}

func (parent *scope) adopt(child *scope) {
	if parent.hasChild(child) {
		return
	}
	parent.birthOrder = append(parent.birthOrder, child)
	parent.children[child.name] = child
}
func (parent *scope) hasChild(child *scope) bool {
	for _, ordered := range parent.birthOrder {
		if ordered.name == child.name && ordered.title == child.title {
			return true
		}
	}
	return false
}

func (self *scope) registerReset(action *action) {
	self.resets[action.name] = action
}

func (self *scope) visited() bool {
	return self.panicked || self.child >= len(self.birthOrder)
}

func (parent *scope) visit() {
	defer parent.exit()
	parent.enter()
	parent.action.Invoke()
	parent.visitChildren()
}
func (parent *scope) enter() {
	parent.reporter.Enter(parent.report)
}
func (parent *scope) visitChildren() {
	if len(parent.birthOrder) == 0 {
		parent.cleanup()
	} else {
		parent.visitChild()
	}
}
func (parent *scope) visitChild() {
	child := parent.birthOrder[parent.child]
	child.visit()
	if child.visited() {
		parent.cleanup()
		parent.child++
	}
}
func (parent *scope) cleanup() {
	for _, reset := range parent.resets {
		reset.Invoke()
	}
}
func (parent *scope) exit() {
	if problem := recover(); problem != nil {
		if strings.HasPrefix(fmt.Sprintf("%v", problem), extraGoTest) {
			panic(problem)
		}
		if problem != failureHalt {
			parent.panicked = true
			parent.reporter.Report(reporting.NewErrorReport(problem))
		}
	}
	parent.reporter.Exit()
}

func newScope(entry *registration, reporter reporting.Reporter) *scope {
	self := new(scope)
	self.reporter = reporter
	self.name = entry.action.name
	self.title = entry.Situation
	self.action = entry.action
	self.children = make(map[string]*scope)
	self.birthOrder = []*scope{}
	self.resets = make(map[string]*action)
	self.report = reporting.NewScopeReport(self.title, self.name)
	return self
}
