// Copyright 2013 Ian Schenck. Use of this source code is governed by
// a license that can be found in the LICENSE file.

/*
	Package envflag adds environment variable flags to the flag package.

	Usage:

	Define flags using envflag.String(), Bool(), Int(), etc. This package
	works nearly the same as the stdlib flag package. Parsing the
	Environment flags is done by calling envflag.Parse()

	It will *not* attempt to parse any normally-defined command-line
	flags. Command-line flags are explicitly left alone and separate.
*/
package envflag

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// VisitAll visits the environment flags in lexicographical order,
// calling fn for each.  It visits all flags, even those not set.
func VisitAll(fn func(*flag.Flag)) {
	EnvironmentFlags.VisitAll(fn)
}

// Visit visits the environment flags in lexicographical order,
// calling fn for each.  It visits only those flags that have been
// set.
func Visit(fn func(*flag.Flag)) {
	EnvironmentFlags.Visit(fn)
}

// Lookup returns the Flag structure of the named environment flag,
// returning nil if none exists.
func Lookup(name string) *flag.Flag {
	return EnvironmentFlags.Lookup(name)
}

// Set sets the value of the named environment flag.
func Set(name, value string) error {
	return EnvironmentFlags.Set(name, value)
}

// BoolVar defines a bool flag with specified name, default value, and
// usage string.  The argument p points to a bool variable in which to
// store the value of the flag.
func BoolVar(p *bool, name string, value bool, usage string) {
	EnvironmentFlags.BoolVar(p, name, value, usage)
}

// Bool defines a bool flag with specified name, default value, and
// usage string.  The return value is the address of a bool variable
// that stores the value of the flag.
func Bool(name string, value bool, usage string) *bool {
	return EnvironmentFlags.Bool(name, value, usage)
}

// IntVar defines an int flag with specified name, default value, and
// usage string.  The argument p points to an int variable in which to
// store the value of the flag.
func IntVar(p *int, name string, value int, usage string) {
	EnvironmentFlags.IntVar(p, name, value, usage)
}

// Int defines an int flag with specified name, default value, and
// usage string.  The return value is the address of an int variable
// that stores the value of the flag.
func Int(name string, value int, usage string) *int {
	return EnvironmentFlags.Int(name, value, usage)
}

// Int64Var defines an int64 flag with specified name, default value,
// and usage string.  The argument p points to an int64 variable in
// which to store the value of the flag.
func Int64Var(p *int64, name string, value int64, usage string) {
	EnvironmentFlags.Int64Var(p, name, value, usage)
}

// Int64 defines an int64 flag with specified name, default value, and
// usage string.  The return value is the address of an int64 variable
// that stores the value of the flag.
func Int64(name string, value int64, usage string) *int64 {
	return EnvironmentFlags.Int64(name, value, usage)
}

// UintVar defines a uint flag with specified name, default value, and
// usage string.  The argument p points to a uint variable in which to
// store the value of the flag.
func UintVar(p *uint, name string, value uint, usage string) {
	EnvironmentFlags.UintVar(p, name, value, usage)
}

// Uint defines a uint flag with specified name, default value, and
// usage string.  The return value is the address of a uint variable
// that stores the value of the flag.
func Uint(name string, value uint, usage string) *uint {
	return EnvironmentFlags.Uint(name, value, usage)
}

// Uint64Var defines a uint64 flag with specified name, default value,
// and usage string.  The argument p points to a uint64 variable in
// which to store the value of the flag.
func Uint64Var(p *uint64, name string, value uint64, usage string) {
	EnvironmentFlags.Uint64Var(p, name, value, usage)
}

// Uint64 defines a uint64 flag with specified name, default value,
// and usage string.  The return value is the address of a uint64
// variable that stores the value of the flag.
func Uint64(name string, value uint64, usage string) *uint64 {
	return EnvironmentFlags.Uint64(name, value, usage)
}

// StringVar defines a string flag with specified name, default value,
// and usage string.  The argument p points to a string variable in
// which to store the value of the flag.
func StringVar(p *string, name string, value string, usage string) {
	EnvironmentFlags.StringVar(p, name, value, usage)
}

// String defines a string flag with specified name, default value,
// and usage string.  The return value is the address of a string
// variable that stores the value of the flag.
func String(name string, value string, usage string) *string {
	return EnvironmentFlags.String(name, value, usage)
}

// Float64Var defines a float64 flag with specified name, default
// value, and usage string.  The argument p points to a float64
// variable in which to store the value of the flag.
func Float64Var(p *float64, name string, value float64, usage string) {
	EnvironmentFlags.Float64Var(p, name, value, usage)
}

// Float64 defines a float64 flag with specified name, default value,
// and usage string.  The return value is the address of a float64
// variable that stores the value of the flag.
func Float64(name string, value float64, usage string) *float64 {
	return EnvironmentFlags.Float64(name, value, usage)
}

// DurationVar defines a time.Duration flag with specified name,
// default value, and usage string.  The argument p points to a
// time.Duration variable in which to store the value of the flag.
func DurationVar(p *time.Duration, name string, value time.Duration, usage string) {
	EnvironmentFlags.DurationVar(p, name, value, usage)
}

// Duration defines a time.Duration flag with specified name, default
// value, and usage string.  The return value is the address of a
// time.Duration variable that stores the value of the flag.
func Duration(name string, value time.Duration, usage string) *time.Duration {
	return EnvironmentFlags.Duration(name, value, usage)
}

// PrintDefaults prints to standard error the default values of all
// defined environment flags.
func PrintDefaults() {
	EnvironmentFlags.PrintDefaults()
}

// Parse parses the environment flags from os.Environ.  Must be called
// after all flags are defined and before flags are accessed by the
// program.
func Parse() {
	env := os.Environ()
	// Clean up and "fake" some flag k/v pairs.
	args := make([]string, 0, len(env))
	for _, value := range env {
		if Lookup(value[:strings.Index(value, "=")]) == nil {
			continue
		}
		args = append(args, fmt.Sprintf("-%s", value))
	}
	EnvironmentFlags.Parse(args)
}

// Parsed returns true if the environment flags have been parsed.
func Parsed() bool {
	return EnvironmentFlags.Parsed()
}

// EnvironmentFlags is the default set of environment flags, parsed
// from os.Environ(). The top-level functions such as BoolVar, Arg,
// and on are wrappers for the methods of EnvironmentFlags.
var EnvironmentFlags = flag.NewFlagSet("environment", flag.ExitOnError)
