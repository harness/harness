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
	"net"

	"github.com/urfave/cli"
	"github.com/Sirupsen/logrus"

	"google.golang.org/grpc"
	grpcCredentials "google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	"github.com/cncd/pipeline/pipeline/rpc/proto"
	"github.com/drone/drone/remote"
	droneserver "github.com/drone/drone/server"
	"github.com/drone/drone/store"
)

func serveGrpc(c *cli.Context, r remote.Remote, v store.Store) error {
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		logrus.Error(err)
		return err
	}
	auther := &authorizer{
		password: c.String("agent-secret"),
	}

	grpcServerOpts := []grpc.ServerOption{
		grpc.StreamInterceptor(auther.streamInterceptor),
		grpc.UnaryInterceptor(auther.unaryIntercaptor),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime: c.Duration("keepalive-min-time"),
		}),
	}

	// Secure gRpc Server communication
	if c.String("grpc-cert-file") != "" && c.String("grpc-key-file") != "" {
		logrus.Debugln("grpc: Configuring TLS")

		creds, err := grpcTLSCreds(c)
		if err != nil {
			return err
		}

		grpcServerOpts = append(grpcServerOpts, grpc.Creds(creds))
	}

	s := grpc.NewServer(grpcServerOpts...)

	ss := new(droneserver.DroneServer)
	ss.Queue = droneserver.Config.Services.Queue
	ss.Logger = droneserver.Config.Services.Logs
	ss.Pubsub = droneserver.Config.Services.Pubsub
	ss.Remote = r
	ss.Store = v
	ss.Host = droneserver.Config.Server.Host
	proto.RegisterDroneServer(s, ss)

	err = s.Serve(lis)
	if err != nil {
		logrus.Error("grpc serve error: %s", err)
		return err
	}
	return nil
}

func grpcTLSCreds(c *cli.Context) (grpcCredentials.TransportCredentials, error) {
	// Load the certificates from disk
	certificate, err := tls.LoadX509KeyPair(c.String("grpc-cert-file"), c.String("grpc-key-file"))
	if err != nil {
		return nil, fmt.Errorf("could not load server key pair: %s", err)
	}

	certPool := x509.NewCertPool()
	if c.String("grpc-ca-cert-file") != "" {
		// Read in custom CA file
		ca, err := ioutil.ReadFile(c.String("grpc-ca-file"))
		if err != nil {
			return nil, fmt.Errorf("could not read ca certificate: %s", err)
		}

		// Append the CA certificate to certPool
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

	mode, err := tlsVerifyMode(c.String("grpc-client-auth"))
	if err != nil {
		return nil, err
	}

	// Create the TLS configuration to pass to the GRPC server
	creds := grpcCredentials.NewTLS(&tls.Config{
		ClientAuth:   *mode,
		Certificates: []tls.Certificate{certificate},
		ClientCAs:    certPool,
	})

	return creds, nil
}

func tlsVerifyMode(mode string) (*tls.ClientAuthType, error) {
	var clientAuthType tls.ClientAuthType

	switch mode {
	case "NoClientCert":
		clientAuthType = tls.NoClientCert
	case "RequestClientCert":
		clientAuthType = tls.RequestClientCert
	case "RequireAnyClientCert":
		clientAuthType = tls.RequireAnyClientCert
	case "VerifyClientCertIfGiven":
		clientAuthType = tls.VerifyClientCertIfGiven
	case "RequireAndVerifyClientCert":
		clientAuthType = tls.RequireAndVerifyClientCert
	default:
		return nil, fmt.Errorf("verifyMode: %s is not a valid mode of: NoClientCert, RequestClientCert, RequireAnyClientCert, VerifyClientCertIfGiven, RequireAndVerifyClientCert", mode)
	}

	return &clientAuthType, nil
}
