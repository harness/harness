// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

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

	// Committer overwrites the git committer used for committing the files (optional, default: actor)
	Committer *Identity
	// Author overwrites the git author used for committing the files (optional, default: committer)
	Author *Identity
}

type CommitFilesResponse struct {
	CommitID string
}

func (c *Client) CommitFiles(ctx context.Context, params *CommitFilesParams) (CommitFilesResponse, error) {
	stream, err := c.commitFilesService.CommitFiles(ctx)
	if err != nil {
		return CommitFilesResponse{}, err
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
				Committer:     mapToRPCIdentityOptional(params.Committer),
			},
		},
	}); err != nil {
		return CommitFilesResponse{}, err
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
			return CommitFilesResponse{}, err
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
				return CommitFilesResponse{}, fmt.Errorf("cannot read buffer: %w", err)
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
				return CommitFilesResponse{}, err
			}
		}
	}

	recv, err := stream.CloseAndRecv()
	if err != nil {
		return CommitFilesResponse{}, err
	}

	return CommitFilesResponse{
		CommitID: recv.CommitId,
	}, nil
}
