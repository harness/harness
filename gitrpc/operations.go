// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"bytes"
	"context"
	"encoding/base64"
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
	Action   FileAction
	Path     string
	Payload  []byte
	Encoding string
	SHA      string
}

// CommitFilesParams holds the data for file operations.
type CommitFilesParams struct {
	WriteParams
	Title     string
	Message   string
	Branch    string
	NewBranch string
	Author    Identity
	Committer Identity
	Actions   []CommitFileAction
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
				Author: &rpc.Identity{
					Name:  params.Author.Name,
					Email: params.Author.Email,
				},
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
		n := 0
		buffer := make([]byte, FileTransferChunkSize)
		reader := io.Reader(bytes.NewReader(action.Payload))
		if action.Encoding == "base64" {
			reader = base64.NewDecoder(base64.StdEncoding, reader)
		}
		for {
			n, err = reader.Read(buffer)
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
