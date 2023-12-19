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

package hook

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// CLICore implements the core of a githook cli. It uses the client and execution timeout
// to perform githook operations as part of a cli.
type CLICore struct {
	client           Client
	executionTimeout time.Duration
}

// NewCLICore returns a new CLICore using the provided client and execution timeout.
func NewCLICore(client Client, executionTimeout time.Duration) *CLICore {
	return &CLICore{
		client:           client,
		executionTimeout: executionTimeout,
	}
}

// PreReceive executes the pre-receive git hook.
func (c *CLICore) PreReceive(ctx context.Context) error {
	refUpdates, err := getUpdatedReferencesFromStdIn()
	if err != nil {
		return fmt.Errorf("failed to read updated references from std in: %w", err)
	}

	in := PreReceiveInput{
		RefUpdates: refUpdates,
	}

	out, err := c.client.PreReceive(ctx, in)

	return handleServerHookOutput(out, err)
}

// Update executes the update git hook.
func (c *CLICore) Update(ctx context.Context, ref string, oldSHA string, newSHA string) error {
	in := UpdateInput{
		RefUpdate: ReferenceUpdate{
			Ref: ref,
			Old: oldSHA,
			New: newSHA,
		},
	}

	out, err := c.client.Update(ctx, in)

	return handleServerHookOutput(out, err)
}

// PostReceive executes the post-receive git hook.
func (c *CLICore) PostReceive(ctx context.Context) error {
	refUpdates, err := getUpdatedReferencesFromStdIn()
	if err != nil {
		return fmt.Errorf("failed to read updated references from std in: %w", err)
	}

	in := PostReceiveInput{
		RefUpdates: refUpdates,
	}

	out, err := c.client.PostReceive(ctx, in)

	return handleServerHookOutput(out, err)
}

//nolint:forbidigo // outputing to CMD as that's where git reads the data
func handleServerHookOutput(out Output, err error) error {
	if err != nil {
		return fmt.Errorf("an error occurred when calling the server: %w", err)
	}

	// print messages before any error
	if len(out.Messages) > 0 {
		// add empty line before and after to make it easier readable
		fmt.Println()
		for _, msg := range out.Messages {
			fmt.Println(msg)
		}
		fmt.Println()
	}

	if out.Error != nil {
		return errors.New(*out.Error)
	}

	return nil
}

// getUpdatedReferencesFromStdIn reads the updated references provided by git from stdin.
// The expected format is "<old-value> SP <new-value> SP <ref-name> LF"
// For more details see https://git-scm.com/docs/githooks#pre-receive
func getUpdatedReferencesFromStdIn() ([]ReferenceUpdate, error) {
	reader := bufio.NewReader(os.Stdin)
	updatedRefs := []ReferenceUpdate{}
	for {
		line, err := reader.ReadString('\n')
		// if end of file is reached, break the loop
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error when reading from standard input - %s\n", err) //nolint:forbidigo // executes as cli.
			return nil, err
		}

		if len(line) == 0 {
			return nil, errors.New("ref data from stdin contains empty line - not expected")
		}

		// splitting line of expected form "<old-value> SP <new-value> SP <ref-name> LF"
		splitGitHookData := strings.Split(line[:len(line)-1], " ")
		if len(splitGitHookData) != 3 {
			return nil, fmt.Errorf("received invalid data format or didn't receive enough parameters - %v",
				splitGitHookData)
		}

		updatedRefs = append(updatedRefs, ReferenceUpdate{
			Old: splitGitHookData[0],
			New: splitGitHookData[1],
			Ref: splitGitHookData[2],
		})
	}

	return updatedRefs, nil
}
