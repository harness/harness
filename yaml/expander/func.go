package expander

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// these are helper functions that bring bash-substitution to the drone yaml file.
// see http://tldp.org/LDP/abs/html/parameter-substitution.html

type substituteFunc func(str, key, val string) string

var substitutors = []substituteFunc{
	substituteQ,
	substitute,
	substitutePrefix,
	substituteSuffix,
	substituteDefault,
	substituteReplace,
	substituteLeft,
	substituteSubstr,
}

// substitute is a helper function that substitutes a simple parameter using
// ${parameter} notation.
func substitute(str, key, val string) string {
	key = fmt.Sprintf("${%s}", key)
	return strings.Replace(str, key, val, -1)
}

// substituteQ is a helper function that substitutes a simple parameter using
// "${parameter}" notation with the escaped value, using %q.
func substituteQ(str, key, val string) string {
	key = fmt.Sprintf(`"${%s}"`, key)
	val = fmt.Sprintf("%q", val)
	return strings.Replace(str, key, val, -1)
}

// substitutePrefix is a helper function that substitutes paramters using
// ${parameter##prefix} notation with the parameter value minus the trimmed prefix.
func substitutePrefix(str, key, val string) string {
	key = fmt.Sprintf("\\${%s##(.+)}", key)
	reg, err := regexp.Compile(key)
	if err != nil {
		return str
	}
	for _, match := range reg.FindAllStringSubmatch(str, -1) {
		if len(match) != 2 {
			continue
		}
		val_ := strings.TrimPrefix(val, match[1])
		str = strings.Replace(str, match[0], val_, -1)
	}
	return str
}

// substituteSuffix is a helper function that substitutes paramters using
// ${parameter%%suffix} notation with the parameter value minus the trimmed suffix.
func substituteSuffix(str, key, val string) string {
	key = fmt.Sprintf("\\${%s%%%%(.+)}", key)
	reg, err := regexp.Compile(key)
	if err != nil {
		return str
	}
	for _, match := range reg.FindAllStringSubmatch(str, -1) {
		if len(match) != 2 {
			continue
		}
		val_ := strings.TrimSuffix(val, match[1])
		str = strings.Replace(str, match[0], val_, -1)
	}
	return str
}

// substituteDefault is a helper function that substitutes paramters using
// ${parameter=default} notation with the parameter value. When empty the
// default value is used.
func substituteDefault(str, key, val string) string {
	key = fmt.Sprintf("\\${%s=(.+)}", key)
	reg, err := regexp.Compile(key)
	if err != nil {
		return str
	}
	for _, match := range reg.FindAllStringSubmatch(str, -1) {
		if len(match) != 2 {
			continue
		}
		if len(val) == 0 {
			str = strings.Replace(str, match[0], match[1], -1)
		} else {
			str = strings.Replace(str, match[0], val, -1)
		}
	}
	return str
}

// substituteReplace is a helper function that substitutes paramters using
// ${parameter/old/new} notation with the parameter value. A find and replace
// is performed before injecting the strings, replacing the old pattern with
// the new value.
func substituteReplace(str, key, val string) string {
	key = fmt.Sprintf("\\${%s/(.+)/(.+)}", key)
	reg, err := regexp.Compile(key)
	if err != nil {
		return str
	}
	for _, match := range reg.FindAllStringSubmatch(str, -1) {
		if len(match) != 3 {
			continue
		}
		with := strings.Replace(val, match[1], match[2], -1)
		str = strings.Replace(str, match[0], with, -1)
	}
	return str
}

// substituteLeft is a helper function that substitutes paramters using
// ${parameter:pos} notation with the parameter value, sliced up to the
// specified position.
func substituteLeft(str, key, val string) string {
	key = fmt.Sprintf("\\${%s:([0-9]*)}", key)
	reg, err := regexp.Compile(key)
	if err != nil {
		return str
	}
	for _, match := range reg.FindAllStringSubmatch(str, -1) {
		if len(match) != 2 {
			continue
		}
		index, err := strconv.Atoi(match[1])
		if err != nil {
			continue // skip
		}
		if index > len(val)-1 {
			continue // skip
		}

		str = strings.Replace(str, match[0], val[:index], -1)
	}
	return str
}

// substituteLeft is a helper function that substitutes paramters using
// ${parameter:pos:len} notation with the parameter value as a substring,
// starting at the specified position for the specified length.
func substituteSubstr(str, key, val string) string {
	key = fmt.Sprintf("\\${%s:([0-9]*):([0-9]*)}", key)
	reg, err := regexp.Compile(key)
	if err != nil {
		return str
	}
	for _, match := range reg.FindAllStringSubmatch(str, -1) {
		if len(match) != 3 {
			continue
		}
		pos, err := strconv.Atoi(match[1])
		if err != nil {
			continue // skip
		}
		length, err := strconv.Atoi(match[2])
		if err != nil {
			continue // skip
		}
		if pos+length > len(val)-1 {
			continue // skip
		}
		str = strings.Replace(str, match[0], val[pos:pos+length], -1)
	}
	return str
}
