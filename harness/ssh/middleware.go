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

package ssh

import (
	"runtime/debug"
	"time"

	"github.com/harness/gitness/app/api/request"

	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Middleware func(ssh.Handler) ssh.Handler

// ChainMiddleware combines multiple middleware into a single ssh.Handler.
func ChainMiddleware(handler ssh.Handler, middlewares ...Middleware) ssh.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- { // Reverse order to maintain correct chaining
		handler = middlewares[i](handler)
	}
	return handler
}

// PanicRecoverMiddleware wraps the SSH handler to recover from panics and log them.
func PanicRecoverMiddleware(next ssh.Handler) ssh.Handler {
	return func(s ssh.Session) {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic and stack trace
				// Get the context and logger
				ctx := s.Context()
				logger := getLogger(ctx)
				logger.Error().Msgf("encountered panic while processing ssh operation: %v\n%s", r, debug.Stack())
				_, _ = s.Write([]byte("Internal server error. Please try again later.\n"))
			}
		}()

		// Call the next handler
		next(s)
	}
}

func HLogAccessLogHandler(next ssh.Handler) ssh.Handler {
	return func(s ssh.Session) {
		start := time.Now()
		user := s.User()
		remoteAddr := s.RemoteAddr()
		command := s.Command()

		// Get the context and logger
		ctx := s.Context()
		logger := getLogger(ctx)
		// Log session start
		logger.Info().
			Str("ssh.user", user).
			Str("ssh.remote", remoteAddr.String()).
			Strs("ssh.command", command).
			Msg("SSH session started")

		// Call the next handler
		next(s)

		// Log session completion
		duration := time.Since(start)
		logger.Info().
			Dur("ssh.elapsed_ms", duration).
			Str("ssh.user", user).
			Msg("SSH session completed")
	}
}

func HLogRequestIDHandler(next ssh.Handler) ssh.Handler {
	return func(s ssh.Session) {
		sshCtx := s.Context() // This is ssh.Context
		reqID := getRequestID(sshCtx.SessionID())
		request.WithRequestIDSSH(sshCtx, reqID)

		log := getLoggerWithRequestID(reqID)
		sshCtx.SetValue(loggerKey, log)

		// continue serving request
		next(s)
	}
}

type PublicKeyMiddleware func(next ssh.PublicKeyHandler) ssh.PublicKeyHandler

func ChainPublicKeyMiddleware(handler ssh.PublicKeyHandler, middlewares ...PublicKeyMiddleware) ssh.PublicKeyHandler {
	for i := len(middlewares) - 1; i >= 0; i-- { // Reverse order for correct chaining
		handler = middlewares[i](handler)
	}
	return handler
}

func LogPublicKeyMiddleware(next ssh.PublicKeyHandler) ssh.PublicKeyHandler {
	return func(ctx ssh.Context, key ssh.PublicKey) bool {
		reqID := getRequestID(ctx.SessionID())
		request.WithRequestIDSSH(ctx, reqID)
		log := getLoggerWithRequestID(reqID)
		start := time.Now()

		log.Info().
			Str("ssh.user", ctx.User()).
			Str("ssh.remote", ctx.RemoteAddr().String()).
			Msg("Public key authentication attempt")

		v := next(ctx, key)
		// Log session completion
		duration := time.Since(start)
		log.Info().
			Dur("ssh.elapsed_ms", duration).
			Str("ssh.user", ctx.User()).
			Msg("Public key authentication attempt completed")
		return v
	}
}

func getLogger(ctx ssh.Context) zerolog.Logger {
	logger, ok := ctx.Value(loggerKey).(zerolog.Logger)
	if !ok {
		logger = log.Logger
	}
	return logger
}
