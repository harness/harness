package goblin

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Reporter interface {
	beginDescribe(string)
	endDescribe()
	begin()
	end()
	failure(*Failure)
	itTook(time.Duration)
	itFailed(string)
	itPassed(string)
	itIsPending(string)
}

type TextFancier interface {
	Red(text string) string
	Gray(text string) string
	Cyan(text string) string
	Green(text string) string
	WithCheck(text string) string
}

type DetailedReporter struct {
	level, failed, passed, pending    int
	failures                          []*Failure
	executionTime, totalExecutionTime time.Duration
	fancy                             TextFancier
}

func (r *DetailedReporter) SetTextFancier(f TextFancier) {
	r.fancy = f
}

type TerminalFancier struct {
}

func (self *TerminalFancier) Red(text string) string {
	return "\033[31m" + text + "\033[0m"
}

func (self *TerminalFancier) Gray(text string) string {
	return "\033[90m" + text + "\033[0m"
}

func (self *TerminalFancier) Cyan(text string) string {
	return "\033[36m" + text + "\033[0m"
}

func (self *TerminalFancier) Green(text string) string {
	return "\033[32m" + text + "\033[0m"
}

func (self *TerminalFancier) WithCheck(text string) string {
	return "\033[32m\u2713\033[0m " + text
}

func (r *DetailedReporter) getSpace() string {
	return strings.Repeat(" ", (r.level+1)*2)
}

func (r *DetailedReporter) failure(failure *Failure) {
	r.failures = append(r.failures, failure)
}

func (r *DetailedReporter) print(text string) {
	fmt.Printf("%v%v\n", r.getSpace(), text)
}

func (r *DetailedReporter) printWithCheck(text string) {
	fmt.Printf("%v%v\n", r.getSpace(), r.fancy.WithCheck(text))
}

func (r *DetailedReporter) beginDescribe(name string) {
	fmt.Println("")
	r.print(name)
	r.level++
}

func (r *DetailedReporter) endDescribe() {
	r.level--
}

func (r *DetailedReporter) itTook(duration time.Duration) {
	r.executionTime = duration
	r.totalExecutionTime += duration
}

func (r *DetailedReporter) itFailed(name string) {
	r.failed++
	r.print(r.fancy.Red(strconv.Itoa(r.failed) + ") " + name))
}

func (r *DetailedReporter) itPassed(name string) {
	r.passed++
	r.printWithCheck(r.fancy.Gray(name))
}

func (r *DetailedReporter) itIsPending(name string) {
	r.pending++
	r.print(r.fancy.Cyan("- " + name))
}

func (r *DetailedReporter) begin() {
}

func (r *DetailedReporter) end() {
	comp := fmt.Sprintf("%d tests complete", r.passed)
	t := fmt.Sprintf("(%d ms)", r.totalExecutionTime/time.Millisecond)

	//fmt.Printf("\n\n \033[32m%d tests complete\033[0m \033[90m(%d ms)\033[0m\n", r.passed, r.totalExecutionTime/time.Millisecond)
	fmt.Printf("\n\n %v %v\n", r.fancy.Green(comp), r.fancy.Gray(t))

	if r.pending > 0 {
		pend := fmt.Sprintf("%d test(s) pending", r.pending)
		fmt.Printf(" %v\n\n", r.fancy.Cyan(pend))
	}

	if len(r.failures) > 0 {
		fmt.Printf("%s \n\n", r.fancy.Red(fmt.Sprintf(" %d tests failed:", len(r.failures))))

	}

	for i, failure := range r.failures {
		fmt.Printf("  %d) %s:\n\n", i+1, failure.testName)
		fmt.Printf("    %s\n", r.fancy.Red(failure.message))
		for _, stackItem := range failure.stack {
			fmt.Printf("    %s\n", r.fancy.Gray(stackItem))
		}
	}
}
