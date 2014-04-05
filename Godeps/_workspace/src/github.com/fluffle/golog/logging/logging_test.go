package logging

import (
	"testing"
)

// Note: the below is deliberately PLACED AT THE TOP OF THIS FILE because
// it is fragile. It ensures the right file:line is logged. Sorry!
func TestLogCorrectLineNumbers(t *testing.T) {
	l, m := newMock(t)
	l.Log(LogError, "Error!")
	// This breaks the mock encapsulation a little, but meh.
	if s := string(m.m[LogError].written); s[20:] != "logging_test.go:11: ERROR Error!\n" {
		t.Errorf("Error incorrectly logged (check line numbers!)")
	}
}

func TestStandardLogging(t *testing.T) {
	l, m := newMock(t)
	l.SetLogLevel(LogError)

	l.Log(4, "Nothing should be logged yet")
	m.ExpectNothing()

	l.Log(LogDebug, "or yet...")
	m.ExpectNothing()

	l.Log(LogInfo, "or yet...")
	m.ExpectNothing()

	l.Log(LogWarn, "or yet!")
	m.ExpectNothing()

	l.Log(LogError, "Error!")
	m.Expect("Error!")
}

func TestAllLoggingLevels(t *testing.T) {
	l, m := newMock(t)

	l.Log(4, "Log to level 4.")
	m.ExpectAt(4, "Log to level 4.")

	l.Debug("Log to debug.")
	m.ExpectAt(LogDebug, "Log to debug.")

	l.Info("Log to info.")
	m.ExpectAt(LogInfo, "Log to info.")

	l.Warn("Log to warning.")
	m.ExpectAt(LogWarn, "Log to warning.")

	l.Error("Log to error.")
	m.ExpectAt(LogError, "Log to error.")

	// recover to track the panic caused by Fatal.
	defer func() { recover() }()
	l.Fatal("Log to fatal.")
	m.ExpectAt(LogFatal, "Log to fatal.")
}
