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
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/harness/gitness/gitrpc/internal/streamio"
	"github.com/harness/gitness/gitrpc/rpc"

	"github.com/rs/zerolog/log"
)

type InfoRefsParams struct {
	ReadParams
	Service     string
	Options     []string // (key, value) pair
	GitProtocol string
}

func (c *Client) GetInfoRefs(ctx context.Context, w io.Writer, params *InfoRefsParams) error {
	if w == nil {
		return errors.New("writer cannot be nil")
	}
	if params == nil {
		return ErrNoParamsProvided
	}
	stream, err := c.httpService.InfoRefs(ctx, &rpc.InfoRefsRequest{
		Base:             mapToRPCReadRequest(params.ReadParams),
		Service:          params.Service,
		GitConfigOptions: params.Options,
		GitProtocol:      params.GitProtocol,
	})
	if err != nil {
		return fmt.Errorf("error initializing GetInfoRefs() stream: %w", err)
	}

	var (
		response *rpc.InfoRefsResponse
	)
	for {
		response, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("GetInfoRefs() error receiving stream bytes: %w", err)
		}
		_, err = w.Write(response.GetData())
		if err != nil {
			return fmt.Errorf("GetInfoRefs() error: %w", err)
		}
	}

	return nil
}

type ServicePackParams struct {
	*ReadParams
	*WriteParams
	Service     string
	GitProtocol string
	Data        io.Reader
	Options     []string // (key, value) pair
}

func (c *Client) ServicePack(ctx context.Context, w io.Writer, params *ServicePackParams) error {
	if w == nil {
		return errors.New("writer cannot be nil")
	}
	if params == nil {
		return ErrNoParamsProvided
	}

	log := log.Ctx(ctx)

	// create request (depends on service whether we need readparams or writeparams)
	// TODO: can we solve this nicer? expose two methods instead?
	request := &rpc.ServicePackRequest{
		Service:          params.Service,
		GitConfigOptions: params.Options,
		GitProtocol:      params.GitProtocol,
	}
	switch params.Service {
	case rpc.ServiceUploadPack:
		if params.ReadParams == nil {
			return errors.New("upload-pack requires ReadParams")
		}
		request.Base = &rpc.ServicePackRequest_ReadBase{
			ReadBase: mapToRPCReadRequest(*params.ReadParams),
		}
	case rpc.ServiceReceivePack:
		if params.WriteParams == nil {
			return errors.New("receive-pack requires WriteParams")
		}
		request.Base = &rpc.ServicePackRequest_WriteBase{
			WriteBase: mapToRPCWriteRequest(*params.WriteParams),
		}
	default:
		return fmt.Errorf("unsupported service provided: %s", params.Service)
	}

	stream, err := c.httpService.ServicePack(ctx)
	if err != nil {
		return err
	}

	log.Debug().Msgf("Start service pack '%s' with options '%v'.",
		params.Service, params.Options)

	// send basic information
	if err = stream.Send(request); err != nil {
		return err
	}

	log.Debug().Msg("Send request stream.")

	// send body as stream
	stdout := streamio.NewWriter(func(p []byte) error {
		return stream.Send(&rpc.ServicePackRequest{
			Data: p,
		})
	})

	_, err = io.Copy(stdout, params.Data)
	if err != nil {
		return fmt.Errorf("PostUploadPack() error copying reader: %w", err)
	}

	log.Debug().Msg("completed sending request stream.")

	if err = stream.CloseSend(); err != nil {
		return fmt.Errorf("PostUploadPack() error closing the stream: %w", err)
	}

	log.Debug().Msg("start receiving response stream.")

	// when we are done with inputs then we should expect
	// git data
	var (
		response *rpc.ServicePackResponse
	)
	for {
		response, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			log.Debug().Msg("received end of response stream.")
			break
		}
		if err != nil {
			return processRPCErrorf(err, "PostUploadPack() error receiving stream bytes")
		}
		if response.GetData() == nil {
			return fmt.Errorf("PostUploadPack() data is nil")
		}

		_, err = w.Write(response.GetData())
		if err != nil {
			return fmt.Errorf("PostUploadPack() error writing response data: %w", err)
		}
	}

	log.Debug().Msg("completed service pack.")

	return nil
}
