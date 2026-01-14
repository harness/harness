// Source: https://github.com/goharbor/harbor

// Copyright 2016 Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package errors

import (
	"fmt"
	"runtime"
	"strings"
)

const maxDepth = 50

type stack []uintptr

func (s *stack) frames() StackFrames {
	var stackFrames StackFrames
	frames := runtime.CallersFrames(*s)
	for {
		frame, next := frames.Next()
		// filter out runtime
		if !strings.Contains(frame.File, "runtime/") {
			stackFrames = append(stackFrames, frame)
		}
		if !next {
			break
		}
	}
	return stackFrames
}

// newStack ...
func newStack() *stack {
	var pcs [maxDepth]uintptr
	n := runtime.Callers(3, pcs[:])
	var st stack = pcs[0:n]
	return &st
}

// StackFrames ...
type StackFrames []runtime.Frame

// Output: <File>:<Line>, <Method>.
func (frames StackFrames) format() string {
	var msg string
	for _, frame := range frames {
		msg += fmt.Sprintf("\n%v:%v, %v", frame.File, frame.Line, frame.Function)
	}
	return msg
}
