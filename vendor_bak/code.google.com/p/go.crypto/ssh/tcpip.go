// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssh

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Listen requests the remote peer open a listening socket
// on addr. Incoming connections will be available by calling
// Accept on the returned net.Listener.
func (c *ClientConn) Listen(n, addr string) (net.Listener, error) {
	laddr, err := net.ResolveTCPAddr(n, addr)
	if err != nil {
		return nil, err
	}
	return c.ListenTCP(laddr)
}

// Automatic port allocation is broken with OpenSSH before 6.0. See
// also https://bugzilla.mindrot.org/show_bug.cgi?id=2017.  In
// particular, OpenSSH 5.9 sends a channelOpenMsg with port number 0,
// rather than the actual port number. This means you can never open
// two different listeners with auto allocated ports. We work around
// this by trying explicit ports until we succeed.

const openSSHPrefix = "OpenSSH_"

var portRandomizer = rand.New(rand.NewSource(time.Now().UnixNano()))

// isBrokenOpenSSHVersion returns true if the given version string
// specifies a version of OpenSSH that is known to have a bug in port
// forwarding.
func isBrokenOpenSSHVersion(versionStr string) bool {
	i := strings.Index(versionStr, openSSHPrefix)
	if i < 0 {
		return false
	}
	i += len(openSSHPrefix)
	j := i
	for ; j < len(versionStr); j++ {
		if versionStr[j] < '0' || versionStr[j] > '9' {
			break
		}
	}
	version, _ := strconv.Atoi(versionStr[i:j])
	return version < 6
}

// autoPortListenWorkaround simulates automatic port allocation by
// trying random ports repeatedly.
func (c *ClientConn) autoPortListenWorkaround(laddr *net.TCPAddr) (net.Listener, error) {
	var sshListener net.Listener
	var err error
	const tries = 10
	for i := 0; i < tries; i++ {
		addr := *laddr
		addr.Port = 1024 + portRandomizer.Intn(60000)
		sshListener, err = c.ListenTCP(&addr)
		if err == nil {
			laddr.Port = addr.Port
			return sshListener, err
		}
	}
	return nil, fmt.Errorf("ssh: listen on random port failed after %d tries: %v", tries, err)
}

// RFC 4254 7.1
type channelForwardMsg struct {
	Message   string
	WantReply bool
	raddr     string
	rport     uint32
}

// ListenTCP requests the remote peer open a listening socket
// on laddr. Incoming connections will be available by calling
// Accept on the returned net.Listener.
func (c *ClientConn) ListenTCP(laddr *net.TCPAddr) (net.Listener, error) {
	if laddr.Port == 0 && isBrokenOpenSSHVersion(c.serverVersion) {
		return c.autoPortListenWorkaround(laddr)
	}

	m := channelForwardMsg{
		"tcpip-forward",
		true, // sendGlobalRequest waits for a reply
		laddr.IP.String(),
		uint32(laddr.Port),
	}
	// send message
	resp, err := c.sendGlobalRequest(m)
	if err != nil {
		return nil, err
	}

	// If the original port was 0, then the remote side will
	// supply a real port number in the response.
	if laddr.Port == 0 {
		port, _, ok := parseUint32(resp.Data)
		if !ok {
			return nil, errors.New("unable to parse response")
		}
		laddr.Port = int(port)
	}

	// Register this forward, using the port number we obtained.
	ch := c.forwardList.add(*laddr)

	return &tcpListener{laddr, c, ch}, nil
}

// forwardList stores a mapping between remote
// forward requests and the tcpListeners.
type forwardList struct {
	sync.Mutex
	entries []forwardEntry
}

// forwardEntry represents an established mapping of a laddr on a
// remote ssh server to a channel connected to a tcpListener.
type forwardEntry struct {
	laddr net.TCPAddr
	c     chan forward
}

// forward represents an incoming forwarded tcpip connection. The
// arguments to add/remove/lookup should be address as specified in
// the original forward-request.
type forward struct {
	c     *clientChan  // the ssh client channel underlying this forward
	raddr *net.TCPAddr // the raddr of the incoming connection
}

func (l *forwardList) add(addr net.TCPAddr) chan forward {
	l.Lock()
	defer l.Unlock()
	f := forwardEntry{
		addr,
		make(chan forward, 1),
	}
	l.entries = append(l.entries, f)
	return f.c
}

// remove removes the forward entry, and the channel feeding its
// listener.
func (l *forwardList) remove(addr net.TCPAddr) {
	l.Lock()
	defer l.Unlock()
	for i, f := range l.entries {
		if addr.IP.Equal(f.laddr.IP) && addr.Port == f.laddr.Port {
			l.entries = append(l.entries[:i], l.entries[i+1:]...)
			close(f.c)
			return
		}
	}
}

// closeAll closes and clears all forwards.
func (l *forwardList) closeAll() {
	l.Lock()
	defer l.Unlock()
	for _, f := range l.entries {
		close(f.c)
	}
	l.entries = nil
}

func (l *forwardList) lookup(addr net.TCPAddr) (chan forward, bool) {
	l.Lock()
	defer l.Unlock()
	for _, f := range l.entries {
		if addr.IP.Equal(f.laddr.IP) && addr.Port == f.laddr.Port {
			return f.c, true
		}
	}
	return nil, false
}

type tcpListener struct {
	laddr *net.TCPAddr

	conn *ClientConn
	in   <-chan forward
}

// Accept waits for and returns the next connection to the listener.
func (l *tcpListener) Accept() (net.Conn, error) {
	s, ok := <-l.in
	if !ok {
		return nil, io.EOF
	}
	return &tcpChanConn{
		tcpChan: &tcpChan{
			clientChan: s.c,
			Reader:     s.c.stdout,
			Writer:     s.c.stdin,
		},
		laddr: l.laddr,
		raddr: s.raddr,
	}, nil
}

// Close closes the listener.
func (l *tcpListener) Close() error {
	m := channelForwardMsg{
		"cancel-tcpip-forward",
		true,
		l.laddr.IP.String(),
		uint32(l.laddr.Port),
	}
	l.conn.forwardList.remove(*l.laddr)
	if _, err := l.conn.sendGlobalRequest(m); err != nil {
		return err
	}
	return nil
}

// Addr returns the listener's network address.
func (l *tcpListener) Addr() net.Addr {
	return l.laddr
}

// Dial initiates a connection to the addr from the remote host.
// The resulting connection has a zero LocalAddr() and RemoteAddr().
func (c *ClientConn) Dial(n, addr string) (net.Conn, error) {
	// Parse the address into host and numeric port.
	host, portString, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	port, err := strconv.ParseUint(portString, 10, 16)
	if err != nil {
		return nil, err
	}
	// Use a zero address for local and remote address.
	zeroAddr := &net.TCPAddr{
		IP:   net.IPv4zero,
		Port: 0,
	}
	ch, err := c.dial(net.IPv4zero.String(), 0, host, int(port))
	if err != nil {
		return nil, err
	}
	return &tcpChanConn{
		tcpChan: ch,
		laddr:   zeroAddr,
		raddr:   zeroAddr,
	}, nil
}

// DialTCP connects to the remote address raddr on the network net,
// which must be "tcp", "tcp4", or "tcp6".  If laddr is not nil, it is used
// as the local address for the connection.
func (c *ClientConn) DialTCP(n string, laddr, raddr *net.TCPAddr) (net.Conn, error) {
	if laddr == nil {
		laddr = &net.TCPAddr{
			IP:   net.IPv4zero,
			Port: 0,
		}
	}
	ch, err := c.dial(laddr.IP.String(), laddr.Port, raddr.IP.String(), raddr.Port)
	if err != nil {
		return nil, err
	}
	return &tcpChanConn{
		tcpChan: ch,
		laddr:   laddr,
		raddr:   raddr,
	}, nil
}

// RFC 4254 7.2
type channelOpenDirectMsg struct {
	ChanType      string
	PeersId       uint32
	PeersWindow   uint32
	MaxPacketSize uint32
	raddr         string
	rport         uint32
	laddr         string
	lport         uint32
}

// dial opens a direct-tcpip connection to the remote server. laddr and raddr are passed as
// strings and are expected to be resolvable at the remote end.
func (c *ClientConn) dial(laddr string, lport int, raddr string, rport int) (*tcpChan, error) {
	ch := c.newChan(c.transport)
	if err := c.transport.writePacket(marshal(msgChannelOpen, channelOpenDirectMsg{
		ChanType:      "direct-tcpip",
		PeersId:       ch.localId,
		PeersWindow:   channelWindowSize,
		MaxPacketSize: channelMaxPacketSize,
		raddr:         raddr,
		rport:         uint32(rport),
		laddr:         laddr,
		lport:         uint32(lport),
	})); err != nil {
		c.chanList.remove(ch.localId)
		return nil, err
	}
	if err := ch.waitForChannelOpenResponse(); err != nil {
		c.chanList.remove(ch.localId)
		return nil, fmt.Errorf("ssh: unable to open direct tcpip connection: %v", err)
	}
	return &tcpChan{
		clientChan: ch,
		Reader:     ch.stdout,
		Writer:     ch.stdin,
	}, nil
}

type tcpChan struct {
	*clientChan // the backing channel
	io.Reader
	io.Writer
}

// tcpChanConn fulfills the net.Conn interface without
// the tcpChan having to hold laddr or raddr directly.
type tcpChanConn struct {
	*tcpChan
	laddr, raddr net.Addr
}

// LocalAddr returns the local network address.
func (t *tcpChanConn) LocalAddr() net.Addr {
	return t.laddr
}

// RemoteAddr returns the remote network address.
func (t *tcpChanConn) RemoteAddr() net.Addr {
	return t.raddr
}

// SetDeadline sets the read and write deadlines associated
// with the connection.
func (t *tcpChanConn) SetDeadline(deadline time.Time) error {
	if err := t.SetReadDeadline(deadline); err != nil {
		return err
	}
	return t.SetWriteDeadline(deadline)
}

// SetReadDeadline sets the read deadline.
// A zero value for t means Read will not time out.
// After the deadline, the error from Read will implement net.Error
// with Timeout() == true.
func (t *tcpChanConn) SetReadDeadline(deadline time.Time) error {
	return errors.New("ssh: tcpChan: deadline not supported")
}

// SetWriteDeadline exists to satisfy the net.Conn interface
// but is not implemented by this type.  It always returns an error.
func (t *tcpChanConn) SetWriteDeadline(deadline time.Time) error {
	return errors.New("ssh: tcpChan: deadline not supported")
}
