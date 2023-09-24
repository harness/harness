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

package gitrpc

import (
	"bytes"
	"context"
	"errors"
	"io"
	"time"

	"github.com/harness/gitness/gitrpc/rpc"
)

type FileAction string

const (
	CreateAction FileAction = "CREATE"
	UpdateAction FileAction = "UPDATE"
	DeleteAction            = "DELETE"
	MoveAction              = "MOVE"
)

func (FileAction) Enum() []interface{} {
	return []interface{}{CreateAction, UpdateAction, DeleteAction, MoveAction}
}

// CommitFileAction holds file operation data.
type CommitFileAction struct {
	Action  FileAction
	Path    string
	Payload []byte
	SHA     string
}

// CommitFilesParams holds the data for file operations.
type CommitFilesParams struct {
	WriteParams
	Title     string
	Message   string
	Branch    string
	NewBranch string
	Actions   []CommitFileAction

	// Committer overwrites the git committer used for committing the files
	// (optional, default: actor)
	Committer *Identity
	// CommitterDate overwrites the git committer date used for committing the files
	// (optional, default: current time on server)
	CommitterDate *time.Time
	// Author overwrites the git author used for committing the files
	// (optional, default: committer)
	Author *Identity
	// AuthorDate overwrites the git author date used for committing the files
	// (optional, default: committer date)
	AuthorDate *time.Time
}

type CommitFilesResponse struct {
	CommitID string
}

func (c *Client) CommitFiles(ctx context.Context, params *CommitFilesParams) (CommitFilesResponse, error) {
	stream, err := c.commitFilesService.CommitFiles(ctx)
	if err != nil {
		return CommitFilesResponse{}, processRPCErrorf(err, "failed to open file stream")
	}

	if err = stream.Send(&rpc.CommitFilesRequest{
		Payload: &rpc.CommitFilesRequest_Header{
			Header: &rpc.CommitFilesRequestHeader{
				Base:          mapToRPCWriteRequest(params.WriteParams),
				BranchName:    params.Branch,
				NewBranchName: params.NewBranch,
				Title:         params.Title,
				Message:       params.Message,
				Author:        mapToRPCIdentityOptional(params.Author),
				AuthorDate:    mapToRPCTimeOptional(params.AuthorDate),
				Committer:     mapToRPCIdentityOptional(params.Committer),
				CommitterDate: mapToRPCTimeOptional(params.CommitterDate),
			},
		},
	}); err != nil {
		return CommitFilesResponse{}, processRPCErrorf(err, "failed to send file headers")
	}

	for _, action := range params.Actions {
		// send headers
		if err = stream.Send(&rpc.CommitFilesRequest{
			Payload: &rpc.CommitFilesRequest_Action{
				Action: &rpc.CommitFilesAction{
					Payload: &rpc.CommitFilesAction_Header{
						Header: &rpc.CommitFilesActionHeader{
							Action: rpc.CommitFilesActionHeader_ActionType(
								rpc.CommitFilesActionHeader_ActionType_value[string(action.Action)]),
							Path: action.Path,
							Sha:  action.SHA,
						},
					},
				},
			},
		}); err != nil {
			return CommitFilesResponse{}, processRPCErrorf(err, "failed to send file action to the stream")
		}

		// send file content
		buffer := make([]byte, FileTransferChunkSize)
		reader := bytes.NewReader(action.Payload)
		for {
			n, err := reader.Read(buffer)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return CommitFilesResponse{}, processRPCErrorf(err, "cannot read buffer")
			}

			if err = stream.Send(&rpc.CommitFilesRequest{
				Payload: &rpc.CommitFilesRequest_Action{
					Action: &rpc.CommitFilesAction{
						Payload: &rpc.CommitFilesAction_Content{
							Content: buffer[:n],
						},
					},
				},
			}); err != nil {
				return CommitFilesResponse{}, processRPCErrorf(err, "failed to send file to the stream")
			}
		}
	}

	recv, err := stream.CloseAndRecv()
	if err != nil {
		return CommitFilesResponse{}, processRPCErrorf(err, "failed to close the stream")
	}

	return CommitFilesResponse{
		CommitID: recv.CommitId,
	}, nil
}
