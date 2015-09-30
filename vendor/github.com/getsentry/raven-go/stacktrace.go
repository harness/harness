// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// Some code from the runtime/debug package of the Go standard library.

package raven

import (
	"bytes"
	"go/build"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// http://sentry.readthedocs.org/en/latest/developer/interfaces/index.html#sentry.interfaces.Stacktrace
type Stacktrace struct {
	// Required
	Frames []*StacktraceFrame `json:"frames"`
}

func (s *Stacktrace) Class() string { return "stacktrace" }

func (s *Stacktrace) Culprit() string {
	for i := len(s.Frames) - 1; i >= 0; i-- {
		frame := s.Frames[i]
		if frame.InApp == true && frame.Module != "" && frame.Function != "" {
			return frame.Module + "." + frame.Function
		}
	}
	return ""
}

type StacktraceFrame struct {
	// At least one required
	Filename string `json:"filename,omitempty"`
	Function string `json:"function,omitempty"`
	Module   string `json:"module,omitempty"`

	// Optional
	Lineno       int      `json:"lineno,omitempty"`
	Colno        int      `json:"colno,omitempty"`
	AbsolutePath string   `json:"abs_path,omitempty"`
	ContextLine  string   `json:"context_line,omitempty"`
	PreContext   []string `json:"pre_context,omitempty"`
	PostContext  []string `json:"post_context,omitempty"`
	InApp        bool     `json:"in_app"`
}

// Intialize and populate a new stacktrace, skipping skip frames.
//
// context is the number of surrounding lines that should be included for context.
// Setting context to 3 would try to get seven lines. Setting context to -1 returns
// one line with no surrounding context, and 0 returns no context.
//
// appPackagePrefixes is a list of prefixes used to check whether a package should
// be considered "in app".
func NewStacktrace(skip int, context int, appPackagePrefixes []string) *Stacktrace {
	var frames []*StacktraceFrame
	for i := 1 + skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		frame := NewStacktraceFrame(pc, file, line, context, appPackagePrefixes)
		if frame != nil {
			frames = append(frames, frame)
		}
	}
	// Sentry wants the frames with the oldest first, so reverse them
	for i, j := 0, len(frames)-1; i < j; i, j = i+1, j-1 {
		frames[i], frames[j] = frames[j], frames[i]
	}
	return &Stacktrace{frames}
}

// Build a single frame using data returned from runtime.Caller.
//
// context is the number of surrounding lines that should be included for context.
// Setting context to 3 would try to get seven lines. Setting context to -1 returns
// one line with no surrounding context, and 0 returns no context.
//
// appPackagePrefixes is a list of prefixes used to check whether a package should
// be considered "in app".
func NewStacktraceFrame(pc uintptr, file string, line, context int, appPackagePrefixes []string) *StacktraceFrame {
	frame := &StacktraceFrame{AbsolutePath: file, Filename: trimPath(file), Lineno: line, InApp: false}
	frame.Module, frame.Function = functionName(pc)

	// `runtime.goexit` is effectively a placeholder that comes from
	// runtime/asm_amd64.s and is meaningless.
	if frame.Module == "runtime" && frame.Function == "goexit" {
		return nil
	}

	if frame.Module == "main" {
		frame.InApp = true
	} else {
		for _, prefix := range appPackagePrefixes {
			if strings.HasPrefix(frame.Module, prefix) && !strings.Contains(frame.Module, "vendor") && !strings.Contains(frame.Module, "third_party") {
				frame.InApp = true
			}
		}
	}

	if context > 0 {
		contextLines, lineIdx := fileContext(file, line, context)
		if len(contextLines) > 0 {
			for i, line := range contextLines {
				switch {
				case i < lineIdx:
					frame.PreContext = append(frame.PreContext, string(line))
				case i == lineIdx:
					frame.ContextLine = string(line)
				default:
					frame.PostContext = append(frame.PostContext, string(line))
				}
			}
		}
	} else if context == -1 {
		contextLine, _ := fileContext(file, line, 0)
		if len(contextLine) > 0 {
			frame.ContextLine = string(contextLine[0])
		}
	}
	return frame
}

// Retrieve the name of the package and function containing the PC.
func functionName(pc uintptr) (pack string, name string) {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return
	}
	name = fn.Name()
	// We get this:
	//	runtime/debug.*T·ptrmethod
	// and want this:
	//  pack = runtime/debug
	//	name = *T.ptrmethod
	if idx := strings.LastIndex(name, "."); idx != -1 {
		pack = name[:idx]
		name = name[idx+1:]
	}
	name = strings.Replace(name, "·", ".", -1)
	return
}

var fileCacheLock sync.Mutex
var fileCache = make(map[string][][]byte)

func fileContext(filename string, line, context int) ([][]byte, int) {
	fileCacheLock.Lock()
	defer fileCacheLock.Unlock()
	lines, ok := fileCache[filename]
	if !ok {
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, 0
		}
		lines = bytes.Split(data, []byte{'\n'})
		fileCache[filename] = lines
	}
	line-- // stack trace lines are 1-indexed
	start := line - context
	var idx int
	if start < 0 {
		start = 0
		idx = line
	} else {
		idx = context
	}
	end := line + context + 1
	if line >= len(lines) {
		return nil, 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	return lines[start:end], idx
}

var trimPaths []string

// Try to trim the GOROOT or GOPATH prefix off of a filename
func trimPath(filename string) string {
	for _, prefix := range trimPaths {
		if trimmed := strings.TrimPrefix(filename, prefix); len(trimmed) < len(filename) {
			return trimmed
		}
	}
	return filename
}

func init() {
	// Collect all source directories, and make sure they
	// end in a trailing "separator"
	for _, prefix := range build.Default.SrcDirs() {
		if prefix[len(prefix)-1] != filepath.Separator {
			prefix += string(filepath.Separator)
		}
		trimPaths = append(trimPaths, prefix)
	}
}
