package convey

func discover(items []interface{}) *registration {
	ensureEnough(items)

	name := parseName(items)
	test := parseGoTest(items)
	action := parseAction(items, test)

	return newRegistration(name, action, test)
}
func ensureEnough(items []interface{}) {
	if len(items) < 2 {
		panic(parseError)
	}
}
func parseName(items []interface{}) string {
	if name, parsed := items[0].(string); parsed {
		return name
	}
	panic(parseError)
}
func parseGoTest(items []interface{}) t {
	if test, parsed := items[1].(t); parsed {
		return test
	}
	return nil
}
func parseAction(items []interface{}, test t) *action {
	var index = 1
	if test != nil {
		index = 2
	}

	if action, parsed := items[index].(func()); parsed {
		return newAction(action)
	}
	if items[index] == nil {
		return newSkippedAction(skipReport)
	}
	panic(parseError)
}

// This interface allows us to pass the *testing.T struct
// throughout the internals of this tool without ever
// having to import the "testing" package.
type t interface {
	Fail()
}

const parseError = "You must provide a name (string), then a *testing.T (if in outermost scope), and then an action (func())."
