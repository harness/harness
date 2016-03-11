package goblin

import (
	"runtime/debug"
	"strings"
)

func ResolveStack(skip int) []string {
	return cleanStack(debug.Stack(), skip)
}

func cleanStack(stack []byte, skip int) []string {
	arrayStack := strings.Split(string(stack), "\n")
	var finalStack []string
	for i := skip; i < len(arrayStack); i++ {
		if strings.Contains(arrayStack[i], ".go") {
			finalStack = append(finalStack, arrayStack[i])
		}
	}
	return finalStack
}
