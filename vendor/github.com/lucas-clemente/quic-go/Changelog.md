# Changelog

## v0.6.0 (unreleased)

- Added `quic.Config` options for maximal flow control windows
- Add a `quic.Config` option for QUIC versions
- Add a `quic.Config` option to request truncation of the connection ID from a server
- Add a `quic.Config` option to configure the source address validation
- Add a `quic.Config` option to configure the handshake timeout
- Add a `quic.Config` option to configure keep-alive
- Implement `net.Conn`-style deadlines for streams
- Remove the `tls.Config` from the `quic.Config`. The `tls.Config` must now be passed to the `Dial` and `Listen` functions as a separate parameter. See the [Godoc](https://godoc.org/github.com/lucas-clemente/quic-go) for details.
- Changed the log level environment variable to only accept strings ("DEBUG", "INFO", "ERROR"), see [the wiki](https://github.com/lucas-clemente/quic-go/wiki/Logging) for more details.
- Rename the `h2quic.QuicRoundTripper` to `h2quic.RoundTripper`
- Changed `h2quic.Server.Serve()` to accept a `net.PacketConn`
- Drop support for Go 1.7.
- Various bugfixes
