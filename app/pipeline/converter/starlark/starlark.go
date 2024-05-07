// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package starlark

import (
	"bytes"
	"errors"

	"github.com/harness/gitness/app/pipeline/file"
	"github.com/harness/gitness/types"

	"github.com/sirupsen/logrus"
	"go.starlark.net/starlark"
)

const (
	separator = "---"
	newline   = "\n"
)

// default limit for generated configuration file size.
const defaultSizeLimit = 1000000

var (
	// ErrMainMissing indicates the starlark script is missing
	// the main method.
	ErrMainMissing = errors.New("starlark: missing main function")

	// ErrMainInvalid indicates the starlark script defines a
	// global variable named main, however, it is not callable.
	ErrMainInvalid = errors.New("starlark: main must be a function")

	// ErrMainReturn indicates the starlark script's main method
	// returns an invalid or unexpected type.
	ErrMainReturn = errors.New("starlark: main returns an invalid type")

	// ErrMaximumSize indicates the starlark script generated a
	// file that exceeds the maximum allowed file size.
	ErrMaximumSize = errors.New("starlark: maximum file size exceeded")

	// ErrCannotLoad indicates the starlark script is attempting to
	// load an external file which is currently restricted.
	ErrCannotLoad = errors.New("starlark: cannot load external scripts")
)

func Parse(
	repo *types.Repository,
	repoIsPublic bool,
	pipeline *types.Pipeline,
	execution *types.Execution,
	file *file.File,
	stepLimit uint64,
	sizeLimit uint64,
) (string, error) {
	thread := &starlark.Thread{
		Name: "drone",
		Load: noLoad,
		Print: func(_ *starlark.Thread, msg string) {
			logrus.WithFields(logrus.Fields{
				"namespace": repo.Path, // TODO: update to just be the space
				"name":      repo.Identifier,
			}).Traceln(msg)
		},
	}
	starlarkFile := file.Data
	starlarkFileName := pipeline.ConfigPath

	globals, err := starlark.ExecFile(thread, starlarkFileName, starlarkFile, nil)
	if err != nil {
		return "", err
	}

	// find the main method in the starlark script and
	// cast to a callable type. If not callable the script
	// is invalid.
	mainVal, ok := globals["main"]
	if !ok {
		return "", ErrMainMissing
	}
	main, ok := mainVal.(starlark.Callable)
	if !ok {
		return "", ErrMainInvalid
	}

	// create the input args and invoke the main method
	// using the input args.
	args := createArgs(repo, pipeline, execution, repoIsPublic)

	// set the maximum number of operations in the script. this
	// mitigates long running scripts.
	if stepLimit == 0 {
		stepLimit = 50000
	}
	thread.SetMaxExecutionSteps(stepLimit)

	// execute the main method in the script.
	mainVal, err = starlark.Call(thread, main, args, nil)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	switch v := mainVal.(type) {
	case *starlark.List:
		for i := 0; i < v.Len(); i++ {
			item := v.Index(i)
			buf.WriteString(separator)
			buf.WriteString(newline)
			if err := write(buf, item); err != nil {
				return "", err
			}
			buf.WriteString(newline)
		}
	case *starlark.Dict:
		if err := write(buf, v); err != nil {
			return "", err
		}
	default:
		return "", ErrMainReturn
	}

	if sizeLimit == 0 {
		sizeLimit = defaultSizeLimit
	}

	// this is a temporary workaround until we
	// implement a LimitWriter.
	if b := buf.Bytes(); uint64(len(b)) > sizeLimit {
		return "", ErrMaximumSize
	}
	return buf.String(), nil
}

func noLoad(_ *starlark.Thread, _ string) (starlark.StringDict, error) {
	return nil, ErrCannotLoad
}
