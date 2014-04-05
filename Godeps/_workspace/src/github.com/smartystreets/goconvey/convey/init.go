package convey

import (
	"os"

	"github.com/smartystreets/goconvey/convey/reporting"
)

func init() {
	suites = newSuiteContext()
}

func buildReporter() reporting.Reporter {
	if testReporter != nil {
		return testReporter

	} else if flagFound(jsonEnabled) {
		return reporting.BuildJsonReporter()

	} else if flagFound(silentEnabled) {
		return reporting.BuildSilentReporter()

	} else if flagFound(verboseEnabled) || flagFound(storyEnabled) {
		return reporting.BuildStoryReporter()

	} else {
		return reporting.BuildDotReporter()

	}
}

// flagFound parses the command line args manually because the go test tool,
// which shares the same process space with this code, already defines
// the -v argument (verbosity) and we can't feed in a custom flag to old-style
// go test packages (like -json, which I would prefer). So, we use the timeout
// flag with a value of -42 to request json output and other negative values
// as needed. My deepest sympothies.
func flagFound(flagValue string) bool {
	for _, arg := range os.Args {
		if arg == flagValue {
			return true
		}
	}
	return false
}

var (
	suites *suiteContext

	// only set by internal tests
	testReporter reporting.Reporter
)

const (
	verboseEnabled = "-test.v=true"

	// Hack! I hope go test *always* supports negative timeouts...
	jsonEnabled   = "-test.timeout=-42s"
	silentEnabled = "-test.timeout=-43s"
	storyEnabled  = "-test.timeout=-44s"
)
