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
	"path/filepath"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/harness/gitness/internal/gitrpc/rpc"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var ErrNoParamsProvided = errors.New("no params provided")

type File struct {
	Name    string
	Base64  bool
	Content []byte // probably base64 encoded data
}

type RepositoryParams struct {
	// TODO: What is it used for?
	Username string
	Name     string
	Branch   string
}

type CreateRepositoryParams struct {
	RepositoryParams
	Files []File
}

type UploadParams struct {
	RepositoryParams
	RepoPath string
	Path     string
}

type AddFilesAndCommitParams struct {
	RepoPath string
	Message  string
	Files    []string
}

type Client struct {
	conn          *grpc.ClientConn
	repoService   rpc.RepositoryServiceClient
	uploadService rpc.UploadServiceClient
}

func InitClient(remoteAddr string) (*Client, error) {
	conn, err := grpc.Dial(remoteAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:          conn,
		repoService:   rpc.NewRepositoryServiceClient(conn),
		uploadService: rpc.NewUploadServiceClient(conn),
	}, nil
}

func (c *Client) CreateRepository(ctx context.Context, params *CreateRepositoryParams) error {
	if params == nil {
		return ErrNoParamsProvided
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	stream, err := c.repoService.CreateRepository(ctx)
	if err != nil {
		return err
	}

	if err = c.sendCreateRepositoryRequest(stream, params); err != nil {
		return err
	}

	if len(params.Files) > 0 {
		for _, file := range params.Files {
			// send filename message
			if err = c.sendCreateRepoFilePath(file.Name, stream); err != nil {
				return err
			}

			// send file content
			buffer := make([]byte, 1024) // todo: chunk size need to be configurable
			reader := bytes.NewReader(file.Content)

			for {
				err = c.process(file.Name, reader, buffer, stream)
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return err
				}
			}
		}
	}
	_, err = stream.CloseAndRecv()
	return err
}

func (c *Client) sendCreateRepositoryRequest(
	stream rpc.RepositoryService_CreateRepositoryClient,
	params *CreateRepositoryParams,
) error {
	req := &rpc.CreateRepositoryRequest{
		Data: &rpc.CreateRepositoryRequest_Repository{
			Repository: &rpc.Repository{
				Owner:         params.Username,
				Name:          params.Name,
				DefaultBranch: params.Branch,
			},
		},
	}

	return stream.Send(req)
}

func (c *Client) sendCreateRepoFilePath(
	filename string,
	stream rpc.RepositoryService_CreateRepositoryClient,
) error {
	req := &rpc.CreateRepositoryRequest{
		Data: &rpc.CreateRepositoryRequest_Filepath{
			Filepath: filename,
		},
	}
	log.Info().Msgf("start sending %v", filename)
	return stream.Send(req)
}

func (c *Client) process(
	filename string,
	reader io.Reader,
	buffer []byte,
	stream rpc.RepositoryService_CreateRepositoryClient) error {
	n, err := reader.Read(buffer)
	if errors.Is(err, io.EOF) {
		log.Info().Msgf("EOF reached for %v", filename)
		err = c.send(buffer[:n], true, stream)
		if err != nil {
			return err
		}

		return io.EOF
	}
	if err != nil {
		return fmt.Errorf("cannot read buffer: %w", err)
	}

	return c.send(buffer[:n], false, stream)
}

func (c *Client) send(buffer []byte, eof bool, stream rpc.RepositoryService_CreateRepositoryClient) error {
	req := &rpc.CreateRepositoryRequest{
		Data: &rpc.CreateRepositoryRequest_Chunk{
			Chunk: &rpc.Chunk{
				Eof:  eof,
				Data: buffer,
			},
		},
	}

	err := stream.Send(req)
	if err != nil {
		err = stream.RecvMsg(nil)
		return status.Errorf(codes.Internal, "cannot send chunk to server: %v", err)
	}
	return nil
}

func (c *Client) UploadFile(ctx context.Context, file *File, params *UploadParams) (string, error) {
	data := file.Content
	if file.Base64 {
		if _, err := base64.StdEncoding.Decode(data, file.Content); err != nil {
			return "", err
		}
	}
	_, err := c.Upload(ctx, params, bytes.NewBuffer(data))
	return file.Name, err
}

func (c *Client) Upload(ctx context.Context, params *UploadParams, reader io.Reader) (string, error) {
	stream, err := c.uploadService.Upload(ctx)
	if err != nil {
		return "", fmt.Errorf("cannot upload file: %w", err)
	}

	req := &rpc.UploadFileRequest{
		Data: &rpc.UploadFileRequest_Info{
			Info: &rpc.FileInfo{
				Username: params.Username,
				Repo:     params.Name,
				Branch:   params.Branch,
				Path:     params.Path,
				RepoPath: params.RepoPath,
				FileType: filepath.Ext(params.Path),
			},
		},
	}

	err = stream.Send(req)
	if err != nil {
		err = stream.RecvMsg(nil)
		return "", status.Errorf(codes.Internal, "cannot send file info to server: %v", err)
	}

	buffer := make([]byte, 1024) // todo: chunk size need to be configurable

	for {
		var n int
		eof := false
		n, err = reader.Read(buffer)
		if errors.Is(err, io.EOF) {
			eof = true
		} else if err != nil {
			return "", fmt.Errorf("cannot read chunk to buffer: %w", err)
		}

		req = &rpc.UploadFileRequest{
			Data: &rpc.UploadFileRequest_Chunk{
				Chunk: &rpc.Chunk{
					Eof:  eof,
					Data: buffer[:n],
				},
			},
		}

		err = stream.Send(req)
		if err != nil {
			err = stream.RecvMsg(nil)
			return "", status.Errorf(codes.Internal, "cannot send chunk to server: %v", err)
		}

		if eof {
			break
		}
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		return "", fmt.Errorf("cannot receive response: %w", err)
	}
	fullPath := res.GetId()
	log.Debug().Msgf("file uploaded with id: %s, size: %d", fullPath, res.GetSize())
	return fullPath, nil
}

func (c *Client) AddFilesAndPush(ctx context.Context, params *AddFilesAndCommitParams) error {
	if params == nil {
		return ErrNoParamsProvided
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, err := c.repoService.AddFilesAndPush(ctx, &rpc.AddFilesAndPushRequest{
		RepoPath: params.RepoPath,
		Message:  params.Message,
		Files:    params.Files,
	})
	return err
}
