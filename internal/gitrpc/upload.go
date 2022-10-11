package gitrpc

import (
	"bytes"
	"errors"
	"io"
	"path/filepath"

	"github.com/harness/gitness/internal/gitrpc/rpc"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxImageSize = 1 << 20

type UploadStore interface {
	Save(filePath string, data bytes.Buffer) (string, error)
}

type uploadService struct {
	rpc.UnimplementedUploadServiceServer
	store   UploadStore
	adapter gitAdapter
}

func newUploadService(adapter gitAdapter, store UploadStore) *uploadService {
	return &uploadService{
		adapter: adapter,
		store:   store,
	}
}

func (s uploadService) Upload(stream rpc.UploadService_UploadServer) error {
	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Unknown, "cannot receive file info")
	}

	fi := req.GetInfo()
	log.Info().Msgf("receive an file request with name %s with file type %s", fi.GetPath(), fi.GetFileType())

	fileData := bytes.Buffer{}
	fileSize := 0

	for {
		log.Info().Msg("waiting to receive more data")

		req, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			log.Print("no more data")
			break
		}
		if err != nil {
			return status.Errorf(codes.Unknown, "cannot receive chunk data: %v", err)
		}

		chunk := req.GetChunkData()
		size := len(chunk)

		log.Printf("received a chunk with size: %d", size)

		fileSize += size
		if fileSize > maxImageSize {
			return status.Errorf(codes.InvalidArgument, "file is too large: %d > %d", fileSize, maxImageSize)
		}
		_, err = fileData.Write(chunk)
		if err != nil {
			return status.Errorf(codes.Internal, "cannot write chunk data: %v", err)
		}
	}
	log.Info().Msgf("saving file %s in repo path %s", fi.GetPath(), fi.GetRepoPath())
	fullPath := filepath.Join(fi.GetRepoPath(), fi.GetPath())
	imageID, err := s.store.Save(fullPath, fileData)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot save file to the store: %v", err)
	}

	res := &rpc.UploadFileResponse{
		Id:   imageID,
		Size: uint32(fileSize),
	}

	err = stream.SendAndClose(res)
	if err != nil {
		return status.Errorf(codes.Unknown, "cannot send response: %v", err)
	}

	log.Info().Msgf("saved file with id: %s, size: %d", imageID, fileSize)
	return nil
}
