package cli

import (
	"flag"
	"fmt"
	"io/ioutil"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const defaultPlaceholder = "value"

// BashCompletionFlag enables bash-completion for all commands and subcommands
var BashCompletionFlag Flag = BoolFlag{
	Name:   "generate-bash-completion",
	Hidden: true,
}

// VersionFlag prints the version for the application
var VersionFlag Flag = BoolFlag{
	Name:  "version, v",
	Usage: "print the version",
}

// HelpFlag prints the help for all commands and subcommands
// Set to the zero value (BoolFlag{}) to disable flag -- keeps subcommand
// unless HideHelp is set to true)
var HelpFlag Flag = BoolFlag{
	Name:  "help, h",
	Usage: "show help",
}

// FlagStringer converts a flag definition to a string. This is used by help
// to display a flag.
var FlagStringer FlagStringFunc = stringifyFlag

// FlagNamePrefixer converts a full flag name and its placeholder into the help
// message flag prefix. This is used by the default FlagStringer.
var FlagNamePrefixer FlagNamePrefixFunc = prefixedNames

// FlagEnvHinter annotates flag help message with the environment variable
// details. This is used by the default FlagStringer.
var FlagEnvHinter FlagEnvHintFunc = withEnvHint

// FlagFileHinter annotates flag help message with the environment variable
// details. This is used by the default FlagStringer.
var FlagFileHinter FlagFileHintFunc = withFileHint

// FlagsByName is a slice of Flag.
type FlagsByName []Flag

func (f FlagsByName) Len() int {
	return len(f)
}

func (f FlagsByName) Less(i, j int) bool {
	return lexicographicLess(f[i].GetName(), f[j].GetName())
}

func (f FlagsByName) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

// Flag is a common interface related to parsing flags in cli.
// For more advanced flag parsing techniques, it is recommended that
// this interface be implemented.
type Flag interface {
	fmt.Stringer
	// Apply Flag settings to the given flag set
	Apply(*flag.FlagSet)
	GetName() string
}

// errorableFlag is an interface that allows us to return errors during apply
// it allows flags defined in this library to return errors in a fashion backwards compatible
// TODO remove in v2 and modify the existing Flag interface to return errors
type errorableFlag interface {
	Flag

	ApplyWithError(*flag.FlagSet) error
}

func flagSet(name string, flags []Flag) (*flag.FlagSet, error) {
	set := flag.NewFlagSet(name, flag.ContinueOnError)

	for _, f := range flags {
		//TODO remove in v2 when errorableFlag is removed
		if ef, ok := f.(errorableFlag); ok {
			if err := ef.ApplyWithError(set); err != nil {
				return nil, err
			}
		} else {
			f.Apply(set)
		}
	}
	return set, nil
}

func eachName(longName string, fn func(string)) {
	parts := strings.Split(longName, ",")
	for _, name := range parts {
		name = strings.Trim(name, " ")
		fn(name)
	}
}

// Generic is a generic parseable type identified by a specific flag
type Generic interface {
	Set(value string) error
	String() string
}

// Apply takes the flagset and calls Set on the generic flag with the value
// provided by the user for parsing by the flag
// Ignores parsing errors
func (f GenericFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

// ApplyWithError takes the flagset and calls Set on the generic flag with the value
// provided by the user for parsing by the flag
func (f GenericFlag) ApplyWithError(set *flag.FlagSet) error {
	val := f.Value
	if fileEnvVal, ok := flagFromFileEnv(f.FilePath, f.EnvVar); ok {
		if err := val.Set(fileEnvVal); err != nil {
			return fmt.Errorf("could not parse %s as value for flag %s: %s", fileEnvVal, f.Name, err)
		}
	}

	eachName(f.Name, func(name string) {
		set.Var(f.Value, name, f.Usage)
	})

	return nil
}

// StringSlice is an opaque type for []string to satisfy flag.Value and flag.Getter
type StringSlice []string

// Set appends the string value to the list of values
func (f *StringSlice) Set(value string) error {
	*f = append(*f, value)
	return nil
}

// String returns a readable representation of this value (for usage defaults)
func (f *StringSlice) String() string {
	return fmt.Sprintf("%s", *f)
}

// Value returns the slice of strings set by this flag
func (f *StringSlice) Value() []string {
	return *f
}

// Get returns the slice of strings set by this flag
func (f *StringSlice) Get() interface{} {
	return *f
}

// Apply populates the flag given the flag set and environment
// Ignores errors
func (f StringSliceFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

// ApplyWithError populates the flag given the flag set and environment
func (f StringSliceFlag) ApplyWithError(set *flag.FlagSet) error {
	if envVal, ok := flagFromFileEnv(f.FilePath, f.EnvVar); ok {
		newVal := &StringSlice{}
		for _, s := range strings.Split(envVal, ",") {
			s = strings.TrimSpace(s)
			if err := newVal.Set(s); err != nil {
				return fmt.Errorf("could not parse %s as string value for flag %s: %s", envVal, f.Name, err)
			}
		}
		if f.Value == nil {
			f.Value = newVal
		} else {
			*f.Value = *newVal
		}
	}

	eachName(f.Name, func(name string) {
		if f.Value == nil {
			f.Value = &StringSlice{}
		}
		set.Var(f.Value, name, f.Usage)
	})

	return nil
}

// IntSlice is an opaque type for []int to satisfy flag.Value and flag.Getter
type IntSlice []int

// Set parses the value into an integer and appends it to the list of values
func (f *IntSlice) Set(value string) error {
	tmp, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	*f = append(*f, tmp)
	return nil
}

// String returns a readable representation of this value (for usage defaults)
func (f *IntSlice) String() string {
	return fmt.Sprintf("%#v", *f)
}

// Value returns the slice of ints set by this flag
func (f *IntSlice) Value() []int {
	return *f
}

// Get returns the slice of ints set by this flag
func (f *IntSlice) Get() interface{} {
	return *f
}

// Apply populates the flag given the flag set and environment
// Ignores errors
func (f IntSliceFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

// ApplyWithError populates the flag given the flag set and environment
func (f IntSliceFlag) ApplyWithError(set *flag.FlagSet) error {
	if envVal, ok := flagFromFileEnv(f.FilePath, f.EnvVar); ok {
		newVal := &IntSlice{}
		for _, s := range strings.Split(envVal, ",") {
			s = strings.TrimSpace(s)
			if err := newVal.Set(s); err != nil {
				return fmt.Errorf("could not parse %s as int slice value for flag %s: %s", envVal, f.Name, err)
			}
		}
		if f.Value == nil {
			f.Value = newVal
		} else {
			*f.Value = *newVal
		}
	}

	eachName(f.Name, func(name string) {
		if f.Value == nil {
			f.Value = &IntSlice{}
		}
		set.Var(f.Value, name, f.Usage)
	})

	return nil
}

// Int64Slice is an opaque type for []int to satisfy flag.Value and flag.Getter
type Int64Slice []int64

// Set parses the value into an integer and appends it to the list of values
func (f *Int64Slice) Set(value string) error {
	tmp, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}
	*f = append(*f, tmp)
	return nil
}

// String returns a readable representation of this value (for usage defaults)
func (f *Int64Slice) String() string {
	return fmt.Sprintf("%#v", *f)
}

// Value returns the slice of ints set by this flag
func (f *Int64Slice) Value() []int64 {
	return *f
}

// Get returns the slice of ints set by this flag
func (f *Int64Slice) Get() interface{} {
	return *f
}

// Apply populates the flag given the flag set and environment
// Ignores errors
func (f Int64SliceFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

// ApplyWithError populates the flag given the flag set and environment
func (f Int64SliceFlag) ApplyWithError(set *flag.FlagSet) error {
	if envVal, ok := flagFromFileEnv(f.FilePath, f.EnvVar); ok {
		newVal := &Int64Slice{}
		for _, s := range strings.Split(envVal, ",") {
			s = strings.TrimSpace(s)
			if err := newVal.Set(s); err != nil {
				return fmt.Errorf("could not parse %s as int64 slice value for flag %s: %s", envVal, f.Name, err)
			}
		}
		if f.Value == nil {
			f.Value = newVal
		} else {
			*f.Value = *newVal
		}
	}

	eachName(f.Name, func(name string) {
		if f.Value == nil {
			f.Value = &Int64Slice{}
		}
		set.Var(f.Value, name, f.Usage)
	})
	return nil
}

// Apply populates the flag given the flag set and environment
// Ignores errors
func (f BoolFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

// ApplyWithError populates the flag given the flag set and environment
func (f BoolFlag) ApplyWithError(set *flag.FlagSet) error {
	val := false
	if envVal, ok := flagFromFileEnv(f.FilePath, f.EnvVar); ok {
		if envVal == "" {
			val = false
		} else {
			envValBool, err := strconv.ParseBool(envVal)
			if err != nil {
				return fmt.Errorf("could not parse %s as bool value for flag %s: %s", envVal, f.Name, err)
			}
			val = envValBool
		}
	}

	eachName(f.Name, func(name string) {
		if f.Destination != nil {
			set.BoolVar(f.Destination, name, val, f.Usage)
			return
		}
		set.Bool(name, val, f.Usage)
	})

	return nil
}

// Apply populates the flag given the flag set and environment
// Ignores errors
func (f BoolTFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

// ApplyWithError populates the flag given the flag set and environment
func (f BoolTFlag) ApplyWithError(set *flag.FlagSet) error {
	val := true

	if envVal, ok := flagFromFileEnv(f.FilePath, f.EnvVar); ok {
		if envVal == "" {
			val = false
		} else {
			envValBool, err := strconv.ParseBool(envVal)
			if err != nil {
				return fmt.Errorf("could not parse %s as bool value for flag %s: %s", envVal, f.Name, err)
			}
			val = envValBool
		}
	}

	eachName(f.Name, func(name string) {
		if f.Destination != nil {
			set.BoolVar(f.Destination, name, val, f.Usage)
			return
		}
		set.Bool(name, val, f.Usage)
	})

	return nil
}

// Apply populates the flag given the flag set and environment
// Ignores errors
func (f StringFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

// ApplyWithError populates the flag given the flag set and environment
func (f StringFlag) ApplyWithError(set *flag.FlagSet) error {
	if envVal, ok := flagFromFileEnv(f.FilePath, f.EnvVar); ok {
		f.Value = envVal
	}

	eachName(f.Name, func(name string) {
		if f.Destination != nil {
			set.StringVar(f.Destination, name, f.Value, f.Usage)
			return
		}
		set.String(name, f.Value, f.Usage)
	})

	return nil
}

// Apply populates the flag given the flag set and environment
// Ignores errors
func (f IntFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

// ApplyWithError populates the flag given the flag set and environment
func (f IntFlag) ApplyWithError(set *flag.FlagSet) error {
	if envVal, ok := flagFromFileEnv(f.FilePath, f.EnvVar); ok {
		envValInt, err := strconv.ParseInt(envVal, 0, 64)
		if err != nil {
			return fmt.Errorf("could not parse %s as int value for flag %s: %s", envVal, f.Name, err)
		}
		f.Value = int(envValInt)
	}

	eachName(f.Name, func(name string) {
		if f.Destination != nil {
			set.IntVar(f.Destination, name, f.Value, f.Usage)
			return
		}
		set.Int(name, f.Value, f.Usage)
	})

	return nil
}

// Apply populates the flag given the flag set and environment
// Ignores errors
func (f Int64Flag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

// ApplyWithError populates the flag given the flag set and environment
func (f Int64Flag) ApplyWithError(set *flag.FlagSet) error {
	if envVal, ok := flagFromFileEnv(f.FilePath, f.EnvVar); ok {
		envValInt, err := strconv.ParseInt(envVal, 0, 64)
		if err != nil {
			return fmt.Errorf("could not parse %s as int value for flag %s: %s", envVal, f.Name, err)
		}

		f.Value = envValInt
	}

	eachName(f.Name, func(name string) {
		if f.Destination != nil {
			set.Int64Var(f.Destination, name, f.Value, f.Usage)
			return
		}
		set.Int64(name, f.Value, f.Usage)
	})

	return nil
}

// Apply populates the flag given the flag set and environment
// Ignores errors
func (f UintFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

// ApplyWithError populates the flag given the flag set and environment
func (f UintFlag) ApplyWithError(set *flag.FlagSet) error {
	if envVal, ok := flagFromFileEnv(f.FilePath, f.EnvVar); ok {
		envValInt, err := strconv.ParseUint(envVal, 0, 64)
		if err != nil {
			return fmt.Errorf("could not parse %s as uint value for flag %s: %s", envVal, f.Name, err)
		}

		f.Value = uint(envValInt)
	}

	eachName(f.Name, func(name string) {
		if f.Destination != nil {
			set.UintVar(f.Destination, name, f.Value, f.Usage)
			return
		}
		set.Uint(name, f.Value, f.Usage)
	})

	return nil
}

// Apply populates the flag given the flag set and environment
// Ignores errors
func (f Uint64Flag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

// ApplyWithError populates the flag given the flag set and environment
func (f Uint64Flag) ApplyWithError(set *flag.FlagSet) error {
	if envVal, ok := flagFromFileEnv(f.FilePath, f.EnvVar); ok {
		envValInt, err := strconv.ParseUint(envVal, 0, 64)
		if err != nil {
			return fmt.Errorf("could not parse %s as uint64 value for flag %s: %s", envVal, f.Name, err)
		}

		f.Value = uint64(envValInt)
	}

	eachName(f.Name, func(name string) {
		if f.Destination != nil {
			set.Uint64Var(f.Destination, name, f.Value, f.Usage)
			return
		}
		set.Uint64(name, f.Value, f.Usage)
	})

	return nil
}

// Apply populates the flag given the flag set and environment
// Ignores errors
func (f DurationFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

// ApplyWithError populates the flag given the flag set and environment
func (f DurationFlag) ApplyWithError(set *flag.FlagSet) error {
	if envVal, ok := flagFromFileEnv(f.FilePath, f.EnvVar); ok {
		envValDuration, err := time.ParseDuration(envVal)
		if err != nil {
			return fmt.Errorf("could not parse %s as duration for flag %s: %s", envVal, f.Name, err)
		}

		f.Value = envValDuration
	}

	eachName(f.Name, func(name string) {
		if f.Destination != nil {
			set.DurationVar(f.Destination, name, f.Value, f.Usage)
			return
		}
		set.Duration(name, f.Value, f.Usage)
	})

	return nil
}

// Apply populates the flag given the flag set and environment
// Ignores errors
func (f Float64Flag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

// ApplyWithError populates the flag given the flag set and environment
func (f Float64Flag) ApplyWithError(set *flag.FlagSet) error {
	if envVal, ok := flagFromFileEnv(f.FilePath, f.EnvVar); ok {
		envValFloat, err := strconv.ParseFloat(envVal, 10)
		if err != nil {
			return fmt.Errorf("could not parse %s as float64 value for flag %s: %s", envVal, f.Name, err)
		}

		f.Value = float64(envValFloat)
	}

	eachName(f.Name, func(name string) {
		if f.Destination != nil {
			set.Float64Var(f.Destination, name, f.Value, f.Usage)
			return
		}
		set.Float64(name, f.Value, f.Usage)
	})

	return nil
}

func visibleFlags(fl []Flag) []Flag {
	visible := []Flag{}
	for _, flag := range fl {
		field := flagValue(flag).FieldByName("Hidden")
		if !field.IsValid() || !field.Bool() {
			visible = append(visible, flag)
		}
	}
	return visible
}

func prefixFor(name string) (prefix string) {
	if len(name) == 1 {
		prefix = "-"
	} else {
		prefix = "--"
	}

	return
}

// Returns the placeholder, if any, and the unquoted usage string.
func unquoteUsage(usage string) (string, string) {
	for i := 0; i < len(usage); i++ {
		if usage[i] == '`' {
			for j := i + 1; j < len(usage); j++ {
				if usage[j] == '`' {
					name := usage[i+1 : j]
					usage = usage[:i] + name + usage[j+1:]
					return name, usage
				}
			}
			break
		}
	}
	return "", usage
}

func prefixedNames(fullName, placeholder string) string {
	var prefixed string
	parts := strings.Split(fullName, ",")
	for i, name := range parts {
		name = strings.Trim(name, " ")
		prefixed += prefixFor(name) + name
		if placeholder != "" {
			prefixed += " " + placeholder
		}
		if i < len(parts)-1 {
			prefixed += ", "
		}
	}
	return prefixed
}

func withEnvHint(envVar, str string) string {
	envText := ""
	if envVar != "" {
		prefix := "$"
		suffix := ""
		sep := ", $"
		if runtime.GOOS == "windows" {
			prefix = "%"
			suffix = "%"
			sep = "%, %"
		}
		envText = " [" + prefix + strings.Join(strings.Split(envVar, ","), sep) + suffix + "]"
	}
	return str + envText
}

func withFileHint(filePath, str string) string {
	fileText := ""
	if filePath != "" {
		fileText = fmt.Sprintf(" [%s]", filePath)
	}
	return str + fileText
}

func flagValue(f Flag) reflect.Value {
	fv := reflect.ValueOf(f)
	for fv.Kind() == reflect.Ptr {
		fv = reflect.Indirect(fv)
	}
	return fv
}

func stringifyFlag(f Flag) string {
	fv := flagValue(f)

	switch f.(type) {
	case IntSliceFlag:
		return FlagFileHinter(
			fv.FieldByName("FilePath").String(),
			FlagEnvHinter(
				fv.FieldByName("EnvVar").String(),
				stringifyIntSliceFlag(f.(IntSliceFlag)),
			),
		)
	case Int64SliceFlag:
		return FlagFileHinter(
			fv.FieldByName("FilePath").String(),
			FlagEnvHinter(
				fv.FieldByName("EnvVar").String(),
				stringifyInt64SliceFlag(f.(Int64SliceFlag)),
			),
		)
	case StringSliceFlag:
		return FlagFileHinter(
			fv.FieldByName("FilePath").String(),
			FlagEnvHinter(
				fv.FieldByName("EnvVar").String(),
				stringifyStringSliceFlag(f.(StringSliceFlag)),
			),
		)
	}

	placeholder, usage := unquoteUsage(fv.FieldByName("Usage").String())

	needsPlaceholder := false
	defaultValueString := ""

	if val := fv.FieldByName("Value"); val.IsValid() {
		needsPlaceholder = true
		defaultValueString = fmt.Sprintf(" (default: %v)", val.Interface())

		if val.Kind() == reflect.String && val.String() != "" {
			defaultValueString = fmt.Sprintf(" (default: %q)", val.String())
		}
	}

	if defaultValueString == " (default: )" {
		defaultValueString = ""
	}

	if needsPlaceholder && placeholder == "" {
		placeholder = defaultPlaceholder
	}

	usageWithDefault := strings.TrimSpace(usage + defaultValueString)

	return FlagFileHinter(
		fv.FieldByName("FilePath").String(),
		FlagEnvHinter(
			fv.FieldByName("EnvVar").String(),
			FlagNamePrefixer(fv.FieldByName("Name").String(), placeholder)+"\t"+usageWithDefault,
		),
	)
}

func stringifyIntSliceFlag(f IntSliceFlag) string {
	defaultVals := []string{}
	if f.Value != nil && len(f.Value.Value()) > 0 {
		for _, i := range f.Value.Value() {
			defaultVals = append(defaultVals, strconv.Itoa(i))
		}
	}

	return stringifySliceFlag(f.Usage, f.Name, defaultVals)
}

func stringifyInt64SliceFlag(f Int64SliceFlag) string {
	defaultVals := []string{}
	if f.Value != nil && len(f.Value.Value()) > 0 {
		for _, i := range f.Value.Value() {
			defaultVals = append(defaultVals, strconv.FormatInt(i, 10))
		}
	}

	return stringifySliceFlag(f.Usage, f.Name, defaultVals)
}

func stringifyStringSliceFlag(f StringSliceFlag) string {
	defaultVals := []string{}
	if f.Value != nil && len(f.Value.Value()) > 0 {
		for _, s := range f.Value.Value() {
			if len(s) > 0 {
				defaultVals = append(defaultVals, strconv.Quote(s))
			}
		}
	}

	return stringifySliceFlag(f.Usage, f.Name, defaultVals)
}

func stringifySliceFlag(usage, name string, defaultVals []string) string {
	placeholder, usage := unquoteUsage(usage)
	if placeholder == "" {
		placeholder = defaultPlaceholder
	}

	defaultVal := ""
	if len(defaultVals) > 0 {
		defaultVal = fmt.Sprintf(" (default: %s)", strings.Join(defaultVals, ", "))
	}

	usageWithDefault := strings.TrimSpace(usage + defaultVal)
	return FlagNamePrefixer(name, placeholder) + "\t" + usageWithDefault
}

func flagFromFileEnv(filePath, envName string) (val string, ok bool) {
	for _, envVar := range strings.Split(envName, ",") {
		envVar = strings.TrimSpace(envVar)
		if envVal, ok := syscall.Getenv(envVar); ok {
			return envVal, true
		}
	}
	for _, fileVar := range strings.Split(filePath, ",") {
		if data, err := ioutil.ReadFile(fileVar); err == nil {
			return string(data), true
		}
	}
	return "", false
}
