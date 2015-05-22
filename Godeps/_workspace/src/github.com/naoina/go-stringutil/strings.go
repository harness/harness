package stringutil

import (
	"bytes"
	"unicode"
)

// ToUpperCamelCase returns a copy of the string s with all Unicode letters mapped to their camel case.
// It will convert to upper case previous letter of '_' and first letter, and remove letter of '_'.
func ToUpperCamelCase(s string) string {
	if s == "" {
		return ""
	}
	upper := true
	var result bytes.Buffer
	for _, c := range s {
		if c == '_' {
			upper = true
			continue
		}
		if upper {
			result.WriteRune(unicode.ToUpper(c))
			upper = false
			continue
		}
		result.WriteRune(c)
	}
	return result.String()
}

// ToUpperCamelCaseASCII is similar to ToUpperCamelCase, but optimized for
// only the ASCII characters.
// ToUpperCamelCaseASCII is faster than ToUpperCamelCase, but doesn't work if
// contains non-ASCII characters.
func ToUpperCamelCaseASCII(s string) string {
	if s == "" {
		return ""
	}
	upper := true
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '_' {
			upper = true
			continue
		}
		if upper {
			result = append(result, toUpperASCII(c))
			upper = false
			continue
		}
		result = append(result, c)
	}
	return string(result)
}

// ToSnakeCase returns a copy of the string s with all Unicode letters mapped to their snake case.
// It will insert letter of '_' at position of previous letter of uppercase and all
// letters convert to lower case.
func ToSnakeCase(s string) string {
	if s == "" {
		return ""
	}
	var result bytes.Buffer
	for _, c := range s {
		if unicode.IsUpper(c) {
			result.WriteByte('_')
		}
		result.WriteRune(unicode.ToLower(c))
	}
	s = result.String()
	if s[0] == '_' {
		return s[1:]
	}
	return s
}

// ToSnakeCaseASCII is similar to ToSnakeCase, but optimized for only the ASCII
// characters.
// ToSnakeCaseASCII is faster than ToSnakeCase, but doesn't work correctly if
// contains non-ASCII characters.
func ToSnakeCaseASCII(s string) string {
	if s == "" {
		return ""
	}
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUpperASCII(c) {
			result = append(result, '_')
		}
		result = append(result, toLowerASCII(c))
	}
	if result[0] == '_' {
		return string(result[1:])
	}
	return string(result)
}

func isUpperASCII(c byte) bool {
	return 'A' <= c && c <= 'Z'
}

func isLowerASCII(c byte) bool {
	return 'a' <= c && c <= 'z'
}

func toUpperASCII(c byte) byte {
	if isLowerASCII(c) {
		return c - ('a' - 'A')
	}
	return c
}

func toLowerASCII(c byte) byte {
	if isUpperASCII(c) {
		return c + 'a' - 'A'
	}
	return c
}
