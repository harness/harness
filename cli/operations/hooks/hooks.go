// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package hooks

import (
	"bufio"
	"io"
	"os"
	"strings"

	"github.com/harness/gitness/client"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/alecthomas/kingpin.v2"
)

type updatedRefData struct {
	branch    string
	oldCommit string
	newCommit string
}

func Register(app *kingpin.Application, client client.Client) {
	cmd := app.Command("hooks", "manage git server hooks")
	registerUpdate(cmd, client)
	registerPostReceive(cmd, client)
	registerPreReceive(cmd, client)
}

func getUpdatedReferencesFromStdIn() ([]updatedRefData, error) {
	reader := bufio.NewReader(os.Stdin)
	var lines []string
	for {
		line, err := reader.ReadString('\n')
		// if end of file is reached, break the loop
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error().Msgf("Error when reading from standard input - %v", err)
			return nil, err
		}
		lines = append(lines, line)

	}
	var updatedRefs []updatedRefData
	for _, data := range lines {
		splitGitHookData := strings.Split(data, " ")
		if len(splitGitHookData) != 3 {
			return nil, status.Errorf(codes.Unknown, "received invalid data format or didn't receive enough parameters - %v", splitGitHookData)
		}

		updatedRefs = append(updatedRefs, updatedRefData{
			oldCommit: splitGitHookData[0],
			newCommit: splitGitHookData[1],
			branch:    splitGitHookData[2],
		})
	}

	return updatedRefs, nil
}
