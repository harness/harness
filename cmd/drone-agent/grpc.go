// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli"

	"google.golang.org/grpc"
	grpcCredentials "google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

func dialGrpc(c *cli.Context) (*grpc.ClientConn, error) {
	dialOpts := []grpc.DialOption{
		grpc.WithPerRPCCredentials(&credentials{
			username: c.String("username"),
			password: c.String("password"),
		}),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time: c.Duration("keepalive-time"),
			Timeout: c.Duration("keepalive-timeout"),
		}),
	}

	if c.Bool("grpc-tls-enable") {
		log.Debug().Msg("grpc: Configuring TLS")

		// Create the client TLS credentials
		creds, err := grpcTLS(c)
		if err != nil {
			return nil, err
		}

		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	} else {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	}

	return grpc.Dial(c.String("server"), dialOpts...)
}

func grpcTLS(c *cli.Context) (grpcCredentials.TransportCredentials, error) {
	var err error
	certPool := x509.NewCertPool()
	if c.String("grpc-ca-cert-file") != "" {
		// Read in custom CA file
		ca, err := ioutil.ReadFile(c.String("grpc-ca-file"))
		if err != nil {
			return nil, fmt.Errorf("could not read ca certificate: %s", err)
		}

		// Append the CA certificate to CertPool
		if ok := certPool.AppendCertsFromPEM(ca); !ok {
			return nil, fmt.Errorf("failed to append ca certs")
		}
	} else {
		// Load System CA certs
		certPool, err = x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
	}

	creds := grpcCredentials.NewTLS(&tls.Config{
		ServerName: c.String("grpc-servername"),
		RootCAs:    certPool,
	})

	// Pass client cert to server for client verification
	if c.String("grpc-cert-file") != "" && c.String("grpc-key-file") != "" {
		certificate, err := tls.LoadX509KeyPair(c.String("grpc-cert-file"), c.String("grpc-key-file"))
		if err != nil {
			return nil, fmt.Errorf("could not load client key pair: %s", err)
		}

		creds = grpcCredentials.NewTLS(&tls.Config{
			ServerName:   c.String("grpc-servername"),
			Certificates: []tls.Certificate{certificate},
			RootCAs:      certPool,
		})
	}

	return creds, nil
}
