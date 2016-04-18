package flags

import (
	"reflect"
)

// Arg represents a positional argument on the command line.
type Arg struct {
	// The name of the positional argument (used in the help)
	Name string

	// A description of the positional argument (used in the help)
	Description string

	// Whether a positional argument is required
	Required int

	value reflect.Value
	tag   multiTag
}

func (a *Arg) isRemaining() bool {
	return a.value.Type().Kind() == reflect.Slice
}
