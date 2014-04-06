// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssh

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

// extendedDataTypeCode identifies an OpenSSL extended data type. See RFC 4254,
// section 5.2.
type extendedDataTypeCode uint32

const (
	// extendedDataStderr is the extended data type that is used for stderr.
	extendedDataStderr extendedDataTypeCode = 1

	// minPacketLength defines the smallest valid packet
	minPacketLength = 9

	// channelMaxPacketSize defines the maximum packet size advertised in open messages
	channelMaxPacketSize = 1 << 15 // RFC 4253 6.1, minimum 32 KiB

	// channelWindowSize defines the window size advertised in open messages
	channelWindowSize = 64 * channelMaxPacketSize // Like OpenSSH
)

// A Channel is an ordered, reliable, duplex stream that is multiplexed over an
// SSH connection. Channel.Read can return a ChannelRequest as an error.
type Channel interface {
	// Accept accepts the channel creation request.
	Accept() error
	// Reject rejects the channel creation request. After calling this, no
	// other methods on the Channel may be called. If they are then the
	// peer is likely to signal a protocol error and drop the connection.
	Reject(reason RejectionReason, message string) error

	// Read may return a ChannelRequest as an error.
	Read(data []byte) (int, error)
	Write(data []byte) (int, error)
	Close() error

	// Stderr returns an io.Writer that writes to this channel with the
	// extended data type set to stderr.
	Stderr() io.Writer

	// AckRequest either sends an ack or nack to the channel request.
	AckRequest(ok bool) error

	// ChannelType returns the type of the channel, as supplied by the
	// client.
	ChannelType() string
	// ExtraData returns the arbitrary payload for this channel, as supplied
	// by the client. This data is specific to the channel type.
	ExtraData() []byte
}

// ChannelRequest represents a request sent on a channel, outside of the normal
// stream of bytes. It may result from calling Read on a Channel.
type ChannelRequest struct {
	Request   string
	WantReply bool
	Payload   []byte
}

func (c ChannelRequest) Error() string {
	return "ssh: channel request received"
}

// RejectionReason is an enumeration used when rejecting channel creation
// requests. See RFC 4254, section 5.1.
type RejectionReason uint32

const (
	Prohibited RejectionReason = iota + 1
	ConnectionFailed
	UnknownChannelType
	ResourceShortage
)

// String converts the rejection reason to human readable form.
func (r RejectionReason) String() string {
	switch r {
	case Prohibited:
		return "administratively prohibited"
	case ConnectionFailed:
		return "connect failed"
	case UnknownChannelType:
		return "unknown channel type"
	case ResourceShortage:
		return "resource shortage"
	}
	return fmt.Sprintf("unknown reason %d", int(r))
}

type channel struct {
	packetConn        // the underlying transport
	localId, remoteId uint32
	remoteWin         window
	maxPacket         uint32
	isClosed          uint32 // atomic bool, non zero if true
}

func (c *channel) sendWindowAdj(n int) error {
	msg := windowAdjustMsg{
		PeersId:         c.remoteId,
		AdditionalBytes: uint32(n),
	}
	return c.writePacket(marshal(msgChannelWindowAdjust, msg))
}

// sendEOF sends EOF to the remote side. RFC 4254 Section 5.3
func (c *channel) sendEOF() error {
	return c.writePacket(marshal(msgChannelEOF, channelEOFMsg{
		PeersId: c.remoteId,
	}))
}

// sendClose informs the remote side of our intent to close the channel.
func (c *channel) sendClose() error {
	return c.packetConn.writePacket(marshal(msgChannelClose, channelCloseMsg{
		PeersId: c.remoteId,
	}))
}

func (c *channel) sendChannelOpenFailure(reason RejectionReason, message string) error {
	reject := channelOpenFailureMsg{
		PeersId:  c.remoteId,
		Reason:   reason,
		Message:  message,
		Language: "en",
	}
	return c.writePacket(marshal(msgChannelOpenFailure, reject))
}

func (c *channel) writePacket(b []byte) error {
	if c.closed() {
		return io.EOF
	}
	if uint32(len(b)) > c.maxPacket {
		return fmt.Errorf("ssh: cannot write %d bytes, maxPacket is %d bytes", len(b), c.maxPacket)
	}
	return c.packetConn.writePacket(b)
}

func (c *channel) closed() bool {
	return atomic.LoadUint32(&c.isClosed) > 0
}

func (c *channel) setClosed() bool {
	return atomic.CompareAndSwapUint32(&c.isClosed, 0, 1)
}

type serverChan struct {
	channel
	// immutable once created
	chanType  string
	extraData []byte

	serverConn  *ServerConn
	myWindow    uint32
	theyClosed  bool // indicates the close msg has been received from the remote side
	theySentEOF bool
	isDead      uint32
	err         error

	pendingRequests []ChannelRequest
	pendingData     []byte
	head, length    int

	// This lock is inferior to serverConn.lock
	cond *sync.Cond
}

func (c *serverChan) Accept() error {
	c.serverConn.lock.Lock()
	defer c.serverConn.lock.Unlock()

	if c.serverConn.err != nil {
		return c.serverConn.err
	}

	confirm := channelOpenConfirmMsg{
		PeersId:       c.remoteId,
		MyId:          c.localId,
		MyWindow:      c.myWindow,
		MaxPacketSize: c.maxPacket,
	}
	return c.writePacket(marshal(msgChannelOpenConfirm, confirm))
}

func (c *serverChan) Reject(reason RejectionReason, message string) error {
	c.serverConn.lock.Lock()
	defer c.serverConn.lock.Unlock()

	if c.serverConn.err != nil {
		return c.serverConn.err
	}

	return c.sendChannelOpenFailure(reason, message)
}

func (c *serverChan) handlePacket(packet interface{}) {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	switch packet := packet.(type) {
	case *channelRequestMsg:
		req := ChannelRequest{
			Request:   packet.Request,
			WantReply: packet.WantReply,
			Payload:   packet.RequestSpecificData,
		}

		c.pendingRequests = append(c.pendingRequests, req)
		c.cond.Signal()
	case *channelCloseMsg:
		c.theyClosed = true
		c.cond.Signal()
	case *channelEOFMsg:
		c.theySentEOF = true
		c.cond.Signal()
	case *windowAdjustMsg:
		if !c.remoteWin.add(packet.AdditionalBytes) {
			panic("illegal window update")
		}
	default:
		panic("unknown packet type")
	}
}

func (c *serverChan) handleData(data []byte) {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	// The other side should never send us more than our window.
	if len(data)+c.length > len(c.pendingData) {
		// TODO(agl): we should tear down the channel with a protocol
		// error.
		return
	}

	c.myWindow -= uint32(len(data))
	for i := 0; i < 2; i++ {
		tail := c.head + c.length
		if tail >= len(c.pendingData) {
			tail -= len(c.pendingData)
		}
		n := copy(c.pendingData[tail:], data)
		data = data[n:]
		c.length += n
	}

	c.cond.Signal()
}

func (c *serverChan) Stderr() io.Writer {
	return extendedDataChannel{c: c, t: extendedDataStderr}
}

// extendedDataChannel is an io.Writer that writes any data to c as extended
// data of the given type.
type extendedDataChannel struct {
	t extendedDataTypeCode
	c *serverChan
}

func (edc extendedDataChannel) Write(data []byte) (n int, err error) {
	const headerLength = 13 // 1 byte message type, 4 bytes remoteId, 4 bytes extended message type, 4 bytes data length
	c := edc.c
	for len(data) > 0 {
		space := min(c.maxPacket-headerLength, len(data))
		if space, err = c.getWindowSpace(space); err != nil {
			return 0, err
		}
		todo := data
		if uint32(len(todo)) > space {
			todo = todo[:space]
		}

		packet := make([]byte, headerLength+len(todo))
		packet[0] = msgChannelExtendedData
		marshalUint32(packet[1:], c.remoteId)
		marshalUint32(packet[5:], uint32(edc.t))
		marshalUint32(packet[9:], uint32(len(todo)))
		copy(packet[13:], todo)

		if err = c.writePacket(packet); err != nil {
			return
		}

		n += len(todo)
		data = data[len(todo):]
	}

	return
}

func (c *serverChan) Read(data []byte) (n int, err error) {
	n, err, windowAdjustment := c.read(data)

	if windowAdjustment > 0 {
		packet := marshal(msgChannelWindowAdjust, windowAdjustMsg{
			PeersId:         c.remoteId,
			AdditionalBytes: windowAdjustment,
		})
		err = c.writePacket(packet)
	}

	return
}

func (c *serverChan) read(data []byte) (n int, err error, windowAdjustment uint32) {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	if c.err != nil {
		return 0, c.err, 0
	}

	for {
		if c.theySentEOF || c.theyClosed || c.dead() {
			return 0, io.EOF, 0
		}

		if len(c.pendingRequests) > 0 {
			req := c.pendingRequests[0]
			if len(c.pendingRequests) == 1 {
				c.pendingRequests = nil
			} else {
				oldPendingRequests := c.pendingRequests
				c.pendingRequests = make([]ChannelRequest, len(oldPendingRequests)-1)
				copy(c.pendingRequests, oldPendingRequests[1:])
			}

			return 0, req, 0
		}

		if c.length > 0 {
			tail := min(uint32(c.head+c.length), len(c.pendingData))
			n = copy(data, c.pendingData[c.head:tail])
			c.head += n
			c.length -= n
			if c.head == len(c.pendingData) {
				c.head = 0
			}

			windowAdjustment = uint32(len(c.pendingData)-c.length) - c.myWindow
			if windowAdjustment < uint32(len(c.pendingData)/2) {
				windowAdjustment = 0
			}
			c.myWindow += windowAdjustment

			return
		}

		c.cond.Wait()
	}

	panic("unreachable")
}

// getWindowSpace takes, at most, max bytes of space from the peer's window. It
// returns the number of bytes actually reserved.
func (c *serverChan) getWindowSpace(max uint32) (uint32, error) {
	if c.dead() || c.closed() {
		return 0, io.EOF
	}
	return c.remoteWin.reserve(max), nil
}

func (c *serverChan) dead() bool {
	return atomic.LoadUint32(&c.isDead) > 0
}

func (c *serverChan) setDead() {
	atomic.StoreUint32(&c.isDead, 1)
}

func (c *serverChan) Write(data []byte) (n int, err error) {
	const headerLength = 9 // 1 byte message type, 4 bytes remoteId, 4 bytes data length
	for len(data) > 0 {
		space := min(c.maxPacket-headerLength, len(data))
		if space, err = c.getWindowSpace(space); err != nil {
			return 0, err
		}
		todo := data
		if uint32(len(todo)) > space {
			todo = todo[:space]
		}

		packet := make([]byte, headerLength+len(todo))
		packet[0] = msgChannelData
		marshalUint32(packet[1:], c.remoteId)
		marshalUint32(packet[5:], uint32(len(todo)))
		copy(packet[9:], todo)

		if err = c.writePacket(packet); err != nil {
			return
		}

		n += len(todo)
		data = data[len(todo):]
	}

	return
}

// Close signals the intent to close the channel.
func (c *serverChan) Close() error {
	c.serverConn.lock.Lock()
	defer c.serverConn.lock.Unlock()

	if c.serverConn.err != nil {
		return c.serverConn.err
	}

	if !c.setClosed() {
		return errors.New("ssh: channel already closed")
	}
	return c.sendClose()
}

func (c *serverChan) AckRequest(ok bool) error {
	c.serverConn.lock.Lock()
	defer c.serverConn.lock.Unlock()

	if c.serverConn.err != nil {
		return c.serverConn.err
	}

	if !ok {
		ack := channelRequestFailureMsg{
			PeersId: c.remoteId,
		}
		return c.writePacket(marshal(msgChannelFailure, ack))
	}

	ack := channelRequestSuccessMsg{
		PeersId: c.remoteId,
	}
	return c.writePacket(marshal(msgChannelSuccess, ack))
}

func (c *serverChan) ChannelType() string {
	return c.chanType
}

func (c *serverChan) ExtraData() []byte {
	return c.extraData
}

// A clientChan represents a single RFC 4254 channel multiplexed
// over a SSH connection.
type clientChan struct {
	channel
	stdin  *chanWriter
	stdout *chanReader
	stderr *chanReader
	msg    chan interface{}
}

// newClientChan returns a partially constructed *clientChan
// using the local id provided. To be usable clientChan.remoteId
// needs to be assigned once known.
func newClientChan(cc packetConn, id uint32) *clientChan {
	c := &clientChan{
		channel: channel{
			packetConn: cc,
			localId:    id,
			remoteWin:  window{Cond: newCond()},
		},
		msg: make(chan interface{}, 16),
	}
	c.stdin = &chanWriter{
		channel: &c.channel,
	}
	c.stdout = &chanReader{
		channel: &c.channel,
		buffer:  newBuffer(),
	}
	c.stderr = &chanReader{
		channel: &c.channel,
		buffer:  newBuffer(),
	}
	return c
}

// waitForChannelOpenResponse, if successful, fills out
// the remoteId and records any initial window advertisement.
func (c *clientChan) waitForChannelOpenResponse() error {
	switch msg := (<-c.msg).(type) {
	case *channelOpenConfirmMsg:
		if msg.MaxPacketSize < minPacketLength || msg.MaxPacketSize > 1<<31 {
			return errors.New("ssh: invalid MaxPacketSize from peer")
		}
		// fixup remoteId field
		c.remoteId = msg.MyId
		c.maxPacket = msg.MaxPacketSize
		c.remoteWin.add(msg.MyWindow)
		return nil
	case *channelOpenFailureMsg:
		return errors.New(safeString(msg.Message))
	}
	return errors.New("ssh: unexpected packet")
}

// Close signals the intent to close the channel.
func (c *clientChan) Close() error {
	if !c.setClosed() {
		return errors.New("ssh: channel already closed")
	}
	c.stdout.eof()
	c.stderr.eof()
	return c.sendClose()
}

// A chanWriter represents the stdin of a remote process.
type chanWriter struct {
	*channel
	// indicates the writer has been closed. eof is owned by the
	// caller of Write/Close.
	eof bool
}

// Write writes data to the remote process's standard input.
func (w *chanWriter) Write(data []byte) (written int, err error) {
	const headerLength = 9 // 1 byte message type, 4 bytes remoteId, 4 bytes data length
	for len(data) > 0 {
		if w.eof || w.closed() {
			err = io.EOF
			return
		}
		// never send more data than maxPacket even if
		// there is sufficient window.
		n := min(w.maxPacket-headerLength, len(data))
		r := w.remoteWin.reserve(n)
		n = r
		remoteId := w.remoteId
		packet := []byte{
			msgChannelData,
			byte(remoteId >> 24), byte(remoteId >> 16), byte(remoteId >> 8), byte(remoteId),
			byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n),
		}
		if err = w.writePacket(append(packet, data[:n]...)); err != nil {
			break
		}
		data = data[n:]
		written += int(n)
	}
	return
}

func min(a uint32, b int) uint32 {
	if a < uint32(b) {
		return a
	}
	return uint32(b)
}

func (w *chanWriter) Close() error {
	w.eof = true
	return w.sendEOF()
}

// A chanReader represents stdout or stderr of a remote process.
type chanReader struct {
	*channel // the channel backing this reader
	*buffer
}

// Read reads data from the remote process's stdout or stderr.
func (r *chanReader) Read(buf []byte) (int, error) {
	n, err := r.buffer.Read(buf)
	if err != nil {
		if err == io.EOF {
			return n, err
		}
		return 0, err
	}
	err = r.sendWindowAdj(n)
	if err == io.EOF && n > 0 {
		// sendWindowAdjust can return io.EOF if the remote peer has
		// closed the connection, however we want to defer forwarding io.EOF to the
		// caller of Read until the buffer has been drained.
		err = nil
	}
	return n, err
}
