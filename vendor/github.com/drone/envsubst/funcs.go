package envsubst

import (
	"path"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// defines a parameter substitution function.
type substituteFunc func(string, ...string) string

// toLen returns the length of string s.
func toLen(s string, args ...string) string {
	return strconv.Itoa(len(s))
}

// toLower returns a copy of the string s with all characters
// mapped to their lower case.
func toLower(s string, args ...string) string {
	return strings.ToLower(s)
}

// toUpper returns a copy of the string s with all characters
// mapped to their upper case.
func toUpper(s string, args ...string) string {
	return strings.ToUpper(s)
}

// toLowerFirst returns a copy of the string s with the first
// character mapped to its lower case.
func toLowerFirst(s string, args ...string) string {
	if s == "" {
		return s
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(r)) + s[n:]
}

// toUpperFirst returns a copy of the string s with the first
// character mapped to its upper case.
func toUpperFirst(s string, args ...string) string {
	if s == "" {
		return s
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

// toDefault returns a copy of the string s if not empty, else
// returns a copy of the first string arugment.
func toDefault(s string, args ...string) string {
	if len(s) == 0 && len(args) == 1 {
		s = args[0]
	}
	return s
}

// toSubstr returns a slice of the string s at the specified
// length and position.
func toSubstr(s string, args ...string) string {
	if len(args) == 0 {
		return s // should never happen
	}

	pos, err := strconv.Atoi(args[0])
	if err != nil {
		// bash returns the string if the position
		// cannot be parsed.
		return s
	}

	if len(args) == 1 {
		if pos < len(s) {
			return s[pos:]
		}
		// if the position exceeds the length of the
		// string an empty string is returned
		return ""
	}

	length, err := strconv.Atoi(args[1])
	if err != nil {
		// bash returns the string if the length
		// cannot be parsed.
		return s
	}

	if pos+length >= len(s) {
		// if the position exceeds the length of the
		// string an empty string is returned
		return ""
	}

	return s[pos : pos+length]
}

// replaceAll returns a copy of the string s with all instances
// of the substring replaced with the replacement string.
func replaceAll(s string, args ...string) string {
	switch len(args) {
	case 0:
		return s
	case 1:
		return strings.Replace(s, args[0], "", -1)
	default:
		return strings.Replace(s, args[0], args[1], -1)
	}
}

// replaceFirst returns a copy of the string s with the first
// instance of the substring replaced with the replacement string.
func replaceFirst(s string, args ...string) string {
	switch len(args) {
	case 0:
		return s
	case 1:
		return strings.Replace(s, args[0], "", 1)
	default:
		return strings.Replace(s, args[0], args[1], 1)
	}
}

// replacePrefix returns a copy of the string s with the matching
// prefix replaced with the replacement string.
func replacePrefix(s string, args ...string) string {
	if len(args) != 2 {
		return s
	}
	if strings.HasPrefix(s, args[0]) {
		return strings.Replace(s, args[0], args[1], 1)
	}
	return s
}

// replaceSuffix returns a copy of the string s with the matching
// suffix replaced with the replacement string.
func replaceSuffix(s string, args ...string) string {
	if len(args) != 2 {
		return s
	}
	if strings.HasSuffix(s, args[0]) {
		s = strings.TrimSuffix(s, args[0])
		s = s + args[1]
	}
	return s
}

// TODO

func trimShortestPrefix(s string, args ...string) string {
	if len(args) != 0 {
		s = trimShortest(s, args[0])
	}
	return s
}

func trimShortestSuffix(s string, args ...string) string {
	if len(args) != 0 {
		r := reverse(s)
		rarg := reverse(args[0])
		s = reverse(trimShortest(r, rarg))
	}
	return s
}

func trimLongestPrefix(s string, args ...string) string {
	if len(args) != 0 {
		s = trimLongest(s, args[0])
	}
	return s
}

func trimLongestSuffix(s string, args ...string) string {
	if len(args) != 0 {
		r := reverse(s)
		rarg := reverse(args[0])
		s = reverse(trimLongest(r, rarg))
	}
	return s
}

func trimShortest(s, arg string) string {
	var shortestMatch string
	for i :=0 ; i < len(s); i++ {
		match, err := path.Match(arg, s[0:len(s)-i])

		if err != nil {
			return s
		}

		if match {
			shortestMatch = s[0:len(s)-i]
		}
	}

	if shortestMatch != "" {
		return strings.TrimPrefix(s, shortestMatch)
	}

	return s
}

func trimLongest(s, arg string) string {
	for i :=0 ; i < len(s); i++ {
		match, err := path.Match(arg, s[0:len(s)-i])

		if err != nil {
			return s
		}

		if match {
			return strings.TrimPrefix(s, s[0:len(s)-i])
		}
	}

	return s
}

func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}
