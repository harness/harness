// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssh

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"code.google.com/p/go.crypto/ssh/terminal"
)

func ExampleListen() {
	// An SSH server is represented by a ServerConfig, which holds
	// certificate details and handles authentication of ServerConns.
	config := &ServerConfig{
		PasswordCallback: func(conn *ServerConn, user, pass string) bool {
			return user == "testuser" && pass == "tiger"
		},
	}

	privateBytes, err := ioutil.ReadFile("id_rsa")
	if err != nil {
		panic("Failed to load private key")
	}

	private, err := ParsePrivateKey(privateBytes)
	if err != nil {
		panic("Failed to parse private key")
	}

	config.AddHostKey(private)

	// Once a ServerConfig has been configured, connections can be
	// accepted.
	listener, err := Listen("tcp", "0.0.0.0:2022", config)
	if err != nil {
		panic("failed to listen for connection")
	}
	sConn, err := listener.Accept()
	if err != nil {
		panic("failed to accept incoming connection")
	}
	if err := sConn.Handshake(); err != nil {
		panic("failed to handshake")
	}

	// A ServerConn multiplexes several channels, which must
	// themselves be Accepted.
	for {
		// Accept reads from the connection, demultiplexes packets
		// to their corresponding channels and returns when a new
		// channel request is seen. Some goroutine must always be
		// calling Accept; otherwise no messages will be forwarded
		// to the channels.
		channel, err := sConn.Accept()
		if err != nil {
			panic("error from Accept")
		}

		// Channels have a type, depending on the application level
		// protocol intended. In the case of a shell, the type is
		// "session" and ServerShell may be used to present a simple
		// terminal interface.
		if channel.ChannelType() != "session" {
			channel.Reject(UnknownChannelType, "unknown channel type")
			continue
		}
		channel.Accept()

		term := terminal.NewTerminal(channel, "> ")
		serverTerm := &ServerTerminal{
			Term:    term,
			Channel: channel,
		}
		go func() {
			defer channel.Close()
			for {
				line, err := serverTerm.ReadLine()
				if err != nil {
					break
				}
				fmt.Println(line)
			}
		}()
	}
}

func ExampleDial() {
	// An SSH client is represented with a ClientConn. Currently only
	// the "password" authentication method is supported.
	//
	// To authenticate with the remote server you must pass at least one
	// implementation of ClientAuth via the Auth field in ClientConfig.
	config := &ClientConfig{
		User: "username",
		Auth: []ClientAuth{
			// ClientAuthPassword wraps a ClientPassword implementation
			// in a type that implements ClientAuth.
			ClientAuthPassword(password("yourpassword")),
		},
	}
	client, err := Dial("tcp", "yourserver.com:22", config)
	if err != nil {
		panic("Failed to dial: " + err.Error())
	}

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := client.NewSession()
	if err != nil {
		panic("Failed to create session: " + err.Error())
	}
	defer session.Close()

	// Once a Session is created, you can execute a single command on
	// the remote side using the Run method.
	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run("/usr/bin/whoami"); err != nil {
		panic("Failed to run: " + err.Error())
	}
	fmt.Println(b.String())
}

func ExampleClientConn_Listen() {
	config := &ClientConfig{
		User: "username",
		Auth: []ClientAuth{
			ClientAuthPassword(password("password")),
		},
	}
	// Dial your ssh server.
	conn, err := Dial("tcp", "localhost:22", config)
	if err != nil {
		log.Fatalf("unable to connect: %s", err)
	}
	defer conn.Close()

	// Request the remote side to open port 8080 on all interfaces.
	l, err := conn.Listen("tcp", "0.0.0.0:8080")
	if err != nil {
		log.Fatalf("unable to register tcp forward: %v", err)
	}
	defer l.Close()

	// Serve HTTP with your SSH server acting as a reverse proxy.
	http.Serve(l, http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(resp, "Hello world!\n")
	}))
}

func ExampleSession_RequestPty() {
	// Create client config
	config := &ClientConfig{
		User: "username",
		Auth: []ClientAuth{
			ClientAuthPassword(password("password")),
		},
	}
	// Connect to ssh server
	conn, err := Dial("tcp", "localhost:22", config)
	if err != nil {
		log.Fatalf("unable to connect: %s", err)
	}
	defer conn.Close()
	// Create a session
	session, err := conn.NewSession()
	if err != nil {
		log.Fatalf("unable to create session: %s", err)
	}
	defer session.Close()
	// Set up terminal modes
	modes := TerminalModes{
		ECHO:          0,     // disable echoing
		TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	// Request pseudo terminal
	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		log.Fatalf("request for pseudo terminal failed: %s", err)
	}
	// Start remote shell
	if err := session.Shell(); err != nil {
		log.Fatalf("failed to start shell: %s", err)
	}
}
