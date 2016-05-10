// Copyright 2012 The Go-MySQL-Driver Authors. All rights reserved.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

// Go MySQL Driver - A MySQL-Driver for Go's database/sql package
//
// The driver should be used via the database/sql package:
//
//  import "database/sql"
//  import _ "github.com/go-sql-driver/mysql"
//
//  db, err := sql.Open("mysql", "user:password@/dbname")
//
// See https://github.com/go-sql-driver/mysql#usage for details
package mysql

import (
	"database/sql"
	"database/sql/driver"
	"net"
)

// This struct is exported to make the driver directly accessible.
// In general the driver is used via the database/sql package.
type MySQLDriver struct{}

// Open new Connection.
// See https://github.com/go-sql-driver/mysql#dsn-data-source-name for how
// the DSN string is formated
func (d *MySQLDriver) Open(dsn string) (driver.Conn, error) {
	var err error

	// New mysqlConn
	mc := &mysqlConn{
		maxPacketAllowed: maxPacketSize,
		maxWriteSize:     maxPacketSize - 1,
	}
	mc.cfg, err = parseDSN(dsn)
	if err != nil {
		return nil, err
	}

	// Connect to Server
	nd := net.Dialer{Timeout: mc.cfg.timeout}
	mc.netConn, err = nd.Dial(mc.cfg.net, mc.cfg.addr)
	if err != nil {
		return nil, err
	}
	mc.buf = newBuffer(mc.netConn)

	// Reading Handshake Initialization Packet
	cipher, err := mc.readInitPacket()
	if err != nil {
		mc.Close()
		return nil, err
	}

	// Send Client Authentication Packet
	if err = mc.writeAuthPacket(cipher); err != nil {
		mc.Close()
		return nil, err
	}

	// Read Result Packet
	err = mc.readResultOK()
	if err != nil {
		// Retry with old authentication method, if allowed
		if mc.cfg.allowOldPasswords && err == errOldPassword {
			if err = mc.writeOldAuthPacket(cipher); err != nil {
				mc.Close()
				return nil, err
			}
			if err = mc.readResultOK(); err != nil {
				mc.Close()
				return nil, err
			}
		} else {
			mc.Close()
			return nil, err
		}

	}

	// Get max allowed packet size
	maxap, err := mc.getSystemVar("max_allowed_packet")
	if err != nil {
		mc.Close()
		return nil, err
	}
	mc.maxPacketAllowed = stringToInt(maxap) - 1
	if mc.maxPacketAllowed < maxPacketSize {
		mc.maxWriteSize = mc.maxPacketAllowed
	}

	// Handle DSN Params
	err = mc.handleParams()
	if err != nil {
		mc.Close()
		return nil, err
	}

	return mc, nil
}

func init() {
	sql.Register("mysql", &MySQLDriver{})
}
