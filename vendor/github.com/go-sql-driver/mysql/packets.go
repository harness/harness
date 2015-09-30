// Go MySQL Driver - A MySQL-Driver for Go's database/sql package
//
// Copyright 2012 The Go-MySQL-Driver Authors. All rights reserved.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

package mysql

import (
	"bytes"
	"crypto/tls"
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"time"
)

// Packets documentation:
// http://dev.mysql.com/doc/internals/en/client-server-protocol.html

// Read packet to buffer 'data'
func (mc *mysqlConn) readPacket() ([]byte, error) {
	// Read packet header
	data, err := mc.buf.readNext(4)
	if err != nil {
		errLog.Print(err.Error())
		mc.Close()
		return nil, driver.ErrBadConn
	}

	// Packet Length [24 bit]
	pktLen := int(uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16)

	if pktLen < 1 {
		errLog.Print(errMalformPkt.Error())
		mc.Close()
		return nil, driver.ErrBadConn
	}

	// Check Packet Sync [8 bit]
	if data[3] != mc.sequence {
		if data[3] > mc.sequence {
			return nil, errPktSyncMul
		} else {
			return nil, errPktSync
		}
	}
	mc.sequence++

	// Read packet body [pktLen bytes]
	if data, err = mc.buf.readNext(pktLen); err == nil {
		if pktLen < maxPacketSize {
			return data, nil
		}

		// Make a copy since data becomes invalid with the next read
		buf := make([]byte, len(data))
		copy(buf, data)

		// More data
		data, err = mc.readPacket()
		if err == nil {
			return append(buf, data...), nil
		}
	}

	// err case
	mc.Close()
	errLog.Print(err.Error())
	return nil, driver.ErrBadConn
}

// Write packet buffer 'data'
// The packet header must be already included
func (mc *mysqlConn) writePacket(data []byte) error {
	if len(data)-4 <= mc.maxWriteSize { // Can send data at once
		// Write packet
		n, err := mc.netConn.Write(data)
		if err == nil && n == len(data) {
			mc.sequence++
			return nil
		}

		// Handle error
		if err == nil { // n != len(data)
			errLog.Print(errMalformPkt.Error())
		} else {
			errLog.Print(err.Error())
		}
		return driver.ErrBadConn
	}

	// Must split packet
	return mc.splitPacket(data)
}

func (mc *mysqlConn) splitPacket(data []byte) error {
	pktLen := len(data) - 4

	if pktLen > mc.maxPacketAllowed {
		return errPktTooLarge
	}

	for pktLen >= maxPacketSize {
		data[0] = 0xff
		data[1] = 0xff
		data[2] = 0xff
		data[3] = mc.sequence

		// Write packet
		n, err := mc.netConn.Write(data[:4+maxPacketSize])
		if err == nil && n == 4+maxPacketSize {
			mc.sequence++
			data = data[maxPacketSize:]
			pktLen -= maxPacketSize
			continue
		}

		// Handle error
		if err == nil { // n != len(data)
			errLog.Print(errMalformPkt.Error())
		} else {
			errLog.Print(err.Error())
		}
		return driver.ErrBadConn
	}

	data[0] = byte(pktLen)
	data[1] = byte(pktLen >> 8)
	data[2] = byte(pktLen >> 16)
	data[3] = mc.sequence
	return mc.writePacket(data)
}

/******************************************************************************
*                           Initialisation Process                            *
******************************************************************************/

// Handshake Initialization Packet
// http://dev.mysql.com/doc/internals/en/connection-phase-packets.html#packet-Protocol::Handshake
func (mc *mysqlConn) readInitPacket() ([]byte, error) {
	data, err := mc.readPacket()
	if err != nil {
		return nil, err
	}

	if data[0] == iERR {
		return nil, mc.handleErrorPacket(data)
	}

	// protocol version [1 byte]
	if data[0] < minProtocolVersion {
		return nil, fmt.Errorf(
			"Unsupported MySQL Protocol Version %d. Protocol Version %d or higher is required",
			data[0],
			minProtocolVersion,
		)
	}

	// server version [null terminated string]
	// connection id [4 bytes]
	pos := 1 + bytes.IndexByte(data[1:], 0x00) + 1 + 4

	// first part of the password cipher [8 bytes]
	cipher := data[pos : pos+8]

	// (filler) always 0x00 [1 byte]
	pos += 8 + 1

	// capability flags (lower 2 bytes) [2 bytes]
	mc.flags = clientFlag(binary.LittleEndian.Uint16(data[pos : pos+2]))
	if mc.flags&clientProtocol41 == 0 {
		return nil, errOldProtocol
	}
	if mc.flags&clientSSL == 0 && mc.cfg.tls != nil {
		return nil, errNoTLS
	}
	pos += 2

	if len(data) > pos {
		// character set [1 byte]
		// status flags [2 bytes]
		// capability flags (upper 2 bytes) [2 bytes]
		// length of auth-plugin-data [1 byte]
		// reserved (all [00]) [10 bytes]
		pos += 1 + 2 + 2 + 1 + 10

		// second part of the password cipher [12? bytes]
		// The documentation is ambiguous about the length.
		// The official Python library uses the fixed length 12
		// which is not documented but seems to work.
		cipher = append(cipher, data[pos:pos+12]...)

		// TODO: Verify string termination
		// EOF if version (>= 5.5.7 and < 5.5.10) or (>= 5.6.0 and < 5.6.2)
		// \NUL otherwise
		//
		//if data[len(data)-1] == 0 {
		//	return
		//}
		//return errMalformPkt
	}

	return cipher, nil
}

// Client Authentication Packet
// http://dev.mysql.com/doc/internals/en/connection-phase-packets.html#packet-Protocol::HandshakeResponse
func (mc *mysqlConn) writeAuthPacket(cipher []byte) error {
	// Adjust client flags based on server support
	clientFlags := clientProtocol41 |
		clientSecureConn |
		clientLongPassword |
		clientTransactions |
		clientLocalFiles |
		mc.flags&clientLongFlag

	if mc.cfg.clientFoundRows {
		clientFlags |= clientFoundRows
	}

	// To enable TLS / SSL
	if mc.cfg.tls != nil {
		clientFlags |= clientSSL
	}

	// User Password
	scrambleBuff := scramblePassword(cipher, []byte(mc.cfg.passwd))

	pktLen := 4 + 4 + 1 + 23 + len(mc.cfg.user) + 1 + 1 + len(scrambleBuff)

	// To specify a db name
	if n := len(mc.cfg.dbname); n > 0 {
		clientFlags |= clientConnectWithDB
		pktLen += n + 1
	}

	// Calculate packet length and get buffer with that size
	data := mc.buf.takeSmallBuffer(pktLen + 4)
	if data == nil {
		// can not take the buffer. Something must be wrong with the connection
		errLog.Print("Busy buffer")
		return driver.ErrBadConn
	}

	// ClientFlags [32 bit]
	data[4] = byte(clientFlags)
	data[5] = byte(clientFlags >> 8)
	data[6] = byte(clientFlags >> 16)
	data[7] = byte(clientFlags >> 24)

	// MaxPacketSize [32 bit] (none)
	data[8] = 0x00
	data[9] = 0x00
	data[10] = 0x00
	data[11] = 0x00

	// Charset [1 byte]
	data[12] = collation_utf8_general_ci

	// SSL Connection Request Packet
	// http://dev.mysql.com/doc/internals/en/connection-phase-packets.html#packet-Protocol::SSLRequest
	if mc.cfg.tls != nil {
		// Packet header  [24bit length + 1 byte sequence]
		data[0] = byte((4 + 4 + 1 + 23))
		data[1] = byte((4 + 4 + 1 + 23) >> 8)
		data[2] = byte((4 + 4 + 1 + 23) >> 16)
		data[3] = mc.sequence

		// Send TLS / SSL request packet
		if err := mc.writePacket(data[:(4+4+1+23)+4]); err != nil {
			return err
		}

		// Switch to TLS
		tlsConn := tls.Client(mc.netConn, mc.cfg.tls)
		if err := tlsConn.Handshake(); err != nil {
			return err
		}
		mc.netConn = tlsConn
		mc.buf.rd = tlsConn
	}

	// Add the packet header  [24bit length + 1 byte sequence]
	data[0] = byte(pktLen)
	data[1] = byte(pktLen >> 8)
	data[2] = byte(pktLen >> 16)
	data[3] = mc.sequence

	// Filler [23 bytes] (all 0x00)
	pos := 13 + 23

	// User [null terminated string]
	if len(mc.cfg.user) > 0 {
		pos += copy(data[pos:], mc.cfg.user)
	}
	data[pos] = 0x00
	pos++

	// ScrambleBuffer [length encoded integer]
	data[pos] = byte(len(scrambleBuff))
	pos += 1 + copy(data[pos+1:], scrambleBuff)

	// Databasename [null terminated string]
	if len(mc.cfg.dbname) > 0 {
		pos += copy(data[pos:], mc.cfg.dbname)
		data[pos] = 0x00
	}

	// Send Auth packet
	return mc.writePacket(data)
}

//  Client old authentication packet
// http://dev.mysql.com/doc/internals/en/connection-phase-packets.html#packet-Protocol::AuthSwitchResponse
func (mc *mysqlConn) writeOldAuthPacket(cipher []byte) error {
	// User password
	scrambleBuff := scrambleOldPassword(cipher, []byte(mc.cfg.passwd))

	// Calculate the packet lenght and add a tailing 0
	pktLen := len(scrambleBuff) + 1
	data := mc.buf.takeSmallBuffer(pktLen + 4)
	if data == nil {
		// can not take the buffer. Something must be wrong with the connection
		errLog.Print("Busy buffer")
		return driver.ErrBadConn
	}

	// Add the packet header  [24bit length + 1 byte sequence]
	data[0] = byte(pktLen)
	data[1] = byte(pktLen >> 8)
	data[2] = byte(pktLen >> 16)
	data[3] = mc.sequence

	// Add the scrambled password [null terminated string]
	copy(data[4:], scrambleBuff)

	return mc.writePacket(data)
}

/******************************************************************************
*                             Command Packets                                 *
******************************************************************************/

func (mc *mysqlConn) writeCommandPacket(command byte) error {
	// Reset Packet Sequence
	mc.sequence = 0

	data := mc.buf.takeSmallBuffer(4 + 1)
	if data == nil {
		// can not take the buffer. Something must be wrong with the connection
		errLog.Print("Busy buffer")
		return driver.ErrBadConn
	}

	// Add the packet header [24bit length + 1 byte sequence]
	data[0] = 0x01 // 1 byte long
	data[1] = 0x00
	data[2] = 0x00
	data[3] = 0x00 // new command, sequence id is always 0

	// Add command byte
	data[4] = command

	// Send CMD packet
	return mc.writePacket(data)
}

func (mc *mysqlConn) writeCommandPacketStr(command byte, arg string) error {
	// Reset Packet Sequence
	mc.sequence = 0

	pktLen := 1 + len(arg)
	data := mc.buf.takeBuffer(pktLen + 4)
	if data == nil {
		// can not take the buffer. Something must be wrong with the connection
		errLog.Print("Busy buffer")
		return driver.ErrBadConn
	}

	// Add the packet header [24bit length + 1 byte sequence]
	data[0] = byte(pktLen)
	data[1] = byte(pktLen >> 8)
	data[2] = byte(pktLen >> 16)
	data[3] = 0x00 // new command, sequence id is always 0

	// Add command byte
	data[4] = command

	// Add arg
	copy(data[5:], arg)

	// Send CMD packet
	return mc.writePacket(data)
}

func (mc *mysqlConn) writeCommandPacketUint32(command byte, arg uint32) error {
	// Reset Packet Sequence
	mc.sequence = 0

	data := mc.buf.takeSmallBuffer(4 + 1 + 4)
	if data == nil {
		// can not take the buffer. Something must be wrong with the connection
		errLog.Print("Busy buffer")
		return driver.ErrBadConn
	}

	// Add the packet header [24bit length + 1 byte sequence]
	data[0] = 0x05 // 5 bytes long
	data[1] = 0x00
	data[2] = 0x00
	data[3] = 0x00 // new command, sequence id is always 0

	// Add command byte
	data[4] = command

	// Add arg [32 bit]
	data[5] = byte(arg)
	data[6] = byte(arg >> 8)
	data[7] = byte(arg >> 16)
	data[8] = byte(arg >> 24)

	// Send CMD packet
	return mc.writePacket(data)
}

/******************************************************************************
*                              Result Packets                                 *
******************************************************************************/

// Returns error if Packet is not an 'Result OK'-Packet
func (mc *mysqlConn) readResultOK() error {
	data, err := mc.readPacket()
	if err == nil {
		// packet indicator
		switch data[0] {

		case iOK:
			return mc.handleOkPacket(data)

		case iEOF:
			// someone is using old_passwords
			return errOldPassword

		default: // Error otherwise
			return mc.handleErrorPacket(data)
		}
	}
	return err
}

// Result Set Header Packet
// http://dev.mysql.com/doc/internals/en/com-query-response.html#packet-ProtocolText::Resultset
func (mc *mysqlConn) readResultSetHeaderPacket() (int, error) {
	data, err := mc.readPacket()
	if err == nil {
		switch data[0] {

		case iOK:
			return 0, mc.handleOkPacket(data)

		case iERR:
			return 0, mc.handleErrorPacket(data)

		case iLocalInFile:
			return 0, mc.handleInFileRequest(string(data[1:]))
		}

		// column count
		num, _, n := readLengthEncodedInteger(data)
		if n-len(data) == 0 {
			return int(num), nil
		}

		return 0, errMalformPkt
	}
	return 0, err
}

// Error Packet
// http://dev.mysql.com/doc/internals/en/generic-response-packets.html#packet-ERR_Packet
func (mc *mysqlConn) handleErrorPacket(data []byte) error {
	if data[0] != iERR {
		return errMalformPkt
	}

	// 0xff [1 byte]

	// Error Number [16 bit uint]
	errno := binary.LittleEndian.Uint16(data[1:3])

	pos := 3

	// SQL State [optional: # + 5bytes string]
	if data[3] == 0x23 {
		//sqlstate := string(data[4 : 4+5])
		pos = 9
	}

	// Error Message [string]
	return &MySQLError{
		Number:  errno,
		Message: string(data[pos:]),
	}
}

// Ok Packet
// http://dev.mysql.com/doc/internals/en/generic-response-packets.html#packet-OK_Packet
func (mc *mysqlConn) handleOkPacket(data []byte) error {
	var n, m int

	// 0x00 [1 byte]

	// Affected rows [Length Coded Binary]
	mc.affectedRows, _, n = readLengthEncodedInteger(data[1:])

	// Insert id [Length Coded Binary]
	mc.insertId, _, m = readLengthEncodedInteger(data[1+n:])

	// server_status [2 bytes]

	// warning count [2 bytes]
	if !mc.strict {
		return nil
	} else {
		pos := 1 + n + m + 2
		if binary.LittleEndian.Uint16(data[pos:pos+2]) > 0 {
			return mc.getWarnings()
		}
		return nil
	}
}

// Read Packets as Field Packets until EOF-Packet or an Error appears
// http://dev.mysql.com/doc/internals/en/com-query-response.html#packet-Protocol::ColumnDefinition41
func (mc *mysqlConn) readColumns(count int) ([]mysqlField, error) {
	columns := make([]mysqlField, count)

	for i := 0; ; i++ {
		data, err := mc.readPacket()
		if err != nil {
			return nil, err
		}

		// EOF Packet
		if data[0] == iEOF && (len(data) == 5 || len(data) == 1) {
			if i == count {
				return columns, nil
			}
			return nil, fmt.Errorf("ColumnsCount mismatch n:%d len:%d", count, len(columns))
		}

		// Catalog
		pos, err := skipLengthEnodedString(data)
		if err != nil {
			return nil, err
		}

		// Database [len coded string]
		n, err := skipLengthEnodedString(data[pos:])
		if err != nil {
			return nil, err
		}
		pos += n

		// Table [len coded string]
		n, err = skipLengthEnodedString(data[pos:])
		if err != nil {
			return nil, err
		}
		pos += n

		// Original table [len coded string]
		n, err = skipLengthEnodedString(data[pos:])
		if err != nil {
			return nil, err
		}
		pos += n

		// Name [len coded string]
		name, _, n, err := readLengthEnodedString(data[pos:])
		if err != nil {
			return nil, err
		}
		columns[i].name = string(name)
		pos += n

		// Original name [len coded string]
		n, err = skipLengthEnodedString(data[pos:])
		if err != nil {
			return nil, err
		}

		// Filler [1 byte]
		// Charset [16 bit uint]
		// Length [32 bit uint]
		pos += n + 1 + 2 + 4

		// Field type [byte]
		columns[i].fieldType = data[pos]
		pos++

		// Flags [16 bit uint]
		columns[i].flags = fieldFlag(binary.LittleEndian.Uint16(data[pos : pos+2]))
		//pos += 2

		// Decimals [8 bit uint]
		//pos++

		// Default value [len coded binary]
		//if pos < len(data) {
		//	defaultVal, _, err = bytesToLengthCodedBinary(data[pos:])
		//}
	}
}

// Read Packets as Field Packets until EOF-Packet or an Error appears
// http://dev.mysql.com/doc/internals/en/com-query-response.html#packet-ProtocolText::ResultsetRow
func (rows *textRows) readRow(dest []driver.Value) error {
	mc := rows.mc

	data, err := mc.readPacket()
	if err != nil {
		return err
	}

	// EOF Packet
	if data[0] == iEOF && len(data) == 5 {
		return io.EOF
	}

	// RowSet Packet
	var n int
	var isNull bool
	pos := 0

	for i := range dest {
		// Read bytes and convert to string
		dest[i], isNull, n, err = readLengthEnodedString(data[pos:])
		pos += n
		if err == nil {
			if !isNull {
				if !mc.parseTime {
					continue
				} else {
					switch rows.columns[i].fieldType {
					case fieldTypeTimestamp, fieldTypeDateTime,
						fieldTypeDate, fieldTypeNewDate:
						dest[i], err = parseDateTime(
							string(dest[i].([]byte)),
							mc.cfg.loc,
						)
						if err == nil {
							continue
						}
					default:
						continue
					}
				}

			} else {
				dest[i] = nil
				continue
			}
		}
		return err // err != nil
	}

	return nil
}

// Reads Packets until EOF-Packet or an Error appears. Returns count of Packets read
func (mc *mysqlConn) readUntilEOF() error {
	for {
		data, err := mc.readPacket()

		// No Err and no EOF Packet
		if err == nil && data[0] != iEOF {
			continue
		}
		return err // Err or EOF
	}
}

/******************************************************************************
*                           Prepared Statements                               *
******************************************************************************/

// Prepare Result Packets
// http://dev.mysql.com/doc/internals/en/com-stmt-prepare-response.html
func (stmt *mysqlStmt) readPrepareResultPacket() (uint16, error) {
	data, err := stmt.mc.readPacket()
	if err == nil {
		// packet indicator [1 byte]
		if data[0] != iOK {
			return 0, stmt.mc.handleErrorPacket(data)
		}

		// statement id [4 bytes]
		stmt.id = binary.LittleEndian.Uint32(data[1:5])

		// Column count [16 bit uint]
		columnCount := binary.LittleEndian.Uint16(data[5:7])

		// Param count [16 bit uint]
		stmt.paramCount = int(binary.LittleEndian.Uint16(data[7:9]))

		// Reserved [8 bit]

		// Warning count [16 bit uint]
		if !stmt.mc.strict {
			return columnCount, nil
		} else {
			// Check for warnings count > 0, only available in MySQL > 4.1
			if len(data) >= 12 && binary.LittleEndian.Uint16(data[10:12]) > 0 {
				return columnCount, stmt.mc.getWarnings()
			}
			return columnCount, nil
		}
	}
	return 0, err
}

// http://dev.mysql.com/doc/internals/en/com-stmt-send-long-data.html
func (stmt *mysqlStmt) writeCommandLongData(paramID int, arg []byte) error {
	maxLen := stmt.mc.maxPacketAllowed - 1
	pktLen := maxLen

	// After the header (bytes 0-3) follows before the data:
	// 1 byte command
	// 4 bytes stmtID
	// 2 bytes paramID
	const dataOffset = 1 + 4 + 2

	// Can not use the write buffer since
	// a) the buffer is too small
	// b) it is in use
	data := make([]byte, 4+1+4+2+len(arg))

	copy(data[4+dataOffset:], arg)

	for argLen := len(arg); argLen > 0; argLen -= pktLen - dataOffset {
		if dataOffset+argLen < maxLen {
			pktLen = dataOffset + argLen
		}

		// Add the packet header [24bit length + 1 byte sequence]
		data[0] = byte(pktLen)
		data[1] = byte(pktLen >> 8)
		data[2] = byte(pktLen >> 16)
		data[3] = 0x00 // mc.sequence

		// Add command byte [1 byte]
		data[4] = comStmtSendLongData

		// Add stmtID [32 bit]
		data[5] = byte(stmt.id)
		data[6] = byte(stmt.id >> 8)
		data[7] = byte(stmt.id >> 16)
		data[8] = byte(stmt.id >> 24)

		// Add paramID [16 bit]
		data[9] = byte(paramID)
		data[10] = byte(paramID >> 8)

		// Send CMD packet
		err := stmt.mc.writePacket(data[:4+pktLen])
		if err == nil {
			data = data[pktLen-dataOffset:]
			continue
		}
		return err

	}

	// Reset Packet Sequence
	stmt.mc.sequence = 0
	return nil
}

// Execute Prepared Statement
// http://dev.mysql.com/doc/internals/en/com-stmt-execute.html
func (stmt *mysqlStmt) writeExecutePacket(args []driver.Value) error {
	if len(args) != stmt.paramCount {
		return fmt.Errorf(
			"Arguments count mismatch (Got: %d Has: %d)",
			len(args),
			stmt.paramCount,
		)
	}

	mc := stmt.mc

	// Reset packet-sequence
	mc.sequence = 0

	var data []byte

	if len(args) == 0 {
		const pktLen = 1 + 4 + 1 + 4
		data = mc.buf.takeBuffer(4 + pktLen)
		if data == nil {
			// can not take the buffer. Something must be wrong with the connection
			errLog.Print("Busy buffer")
			return driver.ErrBadConn
		}

		// packet header [4 bytes]
		data[0] = byte(pktLen)
		data[1] = byte(pktLen >> 8)
		data[2] = byte(pktLen >> 16)
		data[3] = 0x00 // new command, sequence id is always 0
	} else {
		data = mc.buf.takeCompleteBuffer()
		if data == nil {
			// can not take the buffer. Something must be wrong with the connection
			errLog.Print("Busy buffer")
			return driver.ErrBadConn
		}

		// header (bytes 0-3) is added after we know the packet size
	}

	// command [1 byte]
	data[4] = comStmtExecute

	// statement_id [4 bytes]
	data[5] = byte(stmt.id)
	data[6] = byte(stmt.id >> 8)
	data[7] = byte(stmt.id >> 16)
	data[8] = byte(stmt.id >> 24)

	// flags (0: CURSOR_TYPE_NO_CURSOR) [1 byte]
	data[9] = 0x00

	// iteration_count (uint32(1)) [4 bytes]
	data[10] = 0x01
	data[11] = 0x00
	data[12] = 0x00
	data[13] = 0x00

	if len(args) > 0 {
		// NULL-bitmap [(len(args)+7)/8 bytes]
		nullMask := uint64(0)

		pos := 4 + 1 + 4 + 1 + 4 + ((len(args) + 7) >> 3)

		// newParameterBoundFlag 1 [1 byte]
		data[pos] = 0x01
		pos++

		// type of each parameter [len(args)*2 bytes]
		paramTypes := data[pos:]
		pos += (len(args) << 1)

		// value of each parameter [n bytes]
		paramValues := data[pos:pos]
		valuesCap := cap(paramValues)

		for i := range args {
			// build NULL-bitmap
			if args[i] == nil {
				nullMask |= 1 << uint(i)
				paramTypes[i+i] = fieldTypeNULL
				paramTypes[i+i+1] = 0x00
				continue
			}

			// cache types and values
			switch v := args[i].(type) {
			case int64:
				paramTypes[i+i] = fieldTypeLongLong
				paramTypes[i+i+1] = 0x00

				if cap(paramValues)-len(paramValues)-8 >= 0 {
					paramValues = paramValues[:len(paramValues)+8]
					binary.LittleEndian.PutUint64(
						paramValues[len(paramValues)-8:],
						uint64(v),
					)
				} else {
					paramValues = append(paramValues,
						uint64ToBytes(uint64(v))...,
					)
				}

			case float64:
				paramTypes[i+i] = fieldTypeDouble
				paramTypes[i+i+1] = 0x00

				if cap(paramValues)-len(paramValues)-8 >= 0 {
					paramValues = paramValues[:len(paramValues)+8]
					binary.LittleEndian.PutUint64(
						paramValues[len(paramValues)-8:],
						math.Float64bits(v),
					)
				} else {
					paramValues = append(paramValues,
						uint64ToBytes(math.Float64bits(v))...,
					)
				}

			case bool:
				paramTypes[i+i] = fieldTypeTiny
				paramTypes[i+i+1] = 0x00

				if v {
					paramValues = append(paramValues, 0x01)
				} else {
					paramValues = append(paramValues, 0x00)
				}

			case []byte:
				// Common case (non-nil value) first
				if v != nil {
					paramTypes[i+i] = fieldTypeString
					paramTypes[i+i+1] = 0x00

					if len(v) < mc.maxPacketAllowed-pos-len(paramValues)-(len(args)-(i+1))*64 {
						paramValues = appendLengthEncodedInteger(paramValues,
							uint64(len(v)),
						)
						paramValues = append(paramValues, v...)
					} else {
						if err := stmt.writeCommandLongData(i, v); err != nil {
							return err
						}
					}
					continue
				}

				// Handle []byte(nil) as a NULL value
				nullMask |= 1 << uint(i)
				paramTypes[i+i] = fieldTypeNULL
				paramTypes[i+i+1] = 0x00

			case string:
				paramTypes[i+i] = fieldTypeString
				paramTypes[i+i+1] = 0x00

				if len(v) < mc.maxPacketAllowed-pos-len(paramValues)-(len(args)-(i+1))*64 {
					paramValues = appendLengthEncodedInteger(paramValues,
						uint64(len(v)),
					)
					paramValues = append(paramValues, v...)
				} else {
					if err := stmt.writeCommandLongData(i, []byte(v)); err != nil {
						return err
					}
				}

			case time.Time:
				paramTypes[i+i] = fieldTypeString
				paramTypes[i+i+1] = 0x00

				var val []byte
				if v.IsZero() {
					val = []byte("0000-00-00")
				} else {
					val = []byte(v.In(mc.cfg.loc).Format(timeFormat))
				}

				paramValues = appendLengthEncodedInteger(paramValues,
					uint64(len(val)),
				)
				paramValues = append(paramValues, val...)

			default:
				return fmt.Errorf("Can't convert type: %T", args[i])
			}
		}

		// Check if param values exceeded the available buffer
		// In that case we must build the data packet with the new values buffer
		if valuesCap != cap(paramValues) {
			data = append(data[:pos], paramValues...)
			mc.buf.buf = data
		}

		pos += len(paramValues)
		data = data[:pos]

		pktLen := pos - 4

		// packet header [4 bytes]
		data[0] = byte(pktLen)
		data[1] = byte(pktLen >> 8)
		data[2] = byte(pktLen >> 16)
		data[3] = mc.sequence

		// Convert nullMask to bytes
		for i, max := 0, (stmt.paramCount+7)>>3; i < max; i++ {
			data[i+14] = byte(nullMask >> uint(i<<3))
		}
	}

	return mc.writePacket(data)
}

// http://dev.mysql.com/doc/internals/en/binary-protocol-resultset-row.html
func (rows *binaryRows) readRow(dest []driver.Value) error {
	data, err := rows.mc.readPacket()
	if err != nil {
		return err
	}

	// packet indicator [1 byte]
	if data[0] != iOK {
		// EOF Packet
		if data[0] == iEOF && len(data) == 5 {
			return io.EOF
		}

		// Error otherwise
		return rows.mc.handleErrorPacket(data)
	}

	// NULL-bitmap,  [(column-count + 7 + 2) / 8 bytes]
	pos := 1 + (len(dest)+7+2)>>3
	nullMask := data[1:pos]

	for i := range dest {
		// Field is NULL
		// (byte >> bit-pos) % 2 == 1
		if ((nullMask[(i+2)>>3] >> uint((i+2)&7)) & 1) == 1 {
			dest[i] = nil
			continue
		}

		// Convert to byte-coded string
		switch rows.columns[i].fieldType {
		case fieldTypeNULL:
			dest[i] = nil
			continue

		// Numeric Types
		case fieldTypeTiny:
			if rows.columns[i].flags&flagUnsigned != 0 {
				dest[i] = int64(data[pos])
			} else {
				dest[i] = int64(int8(data[pos]))
			}
			pos++
			continue

		case fieldTypeShort, fieldTypeYear:
			if rows.columns[i].flags&flagUnsigned != 0 {
				dest[i] = int64(binary.LittleEndian.Uint16(data[pos : pos+2]))
			} else {
				dest[i] = int64(int16(binary.LittleEndian.Uint16(data[pos : pos+2])))
			}
			pos += 2
			continue

		case fieldTypeInt24, fieldTypeLong:
			if rows.columns[i].flags&flagUnsigned != 0 {
				dest[i] = int64(binary.LittleEndian.Uint32(data[pos : pos+4]))
			} else {
				dest[i] = int64(int32(binary.LittleEndian.Uint32(data[pos : pos+4])))
			}
			pos += 4
			continue

		case fieldTypeLongLong:
			if rows.columns[i].flags&flagUnsigned != 0 {
				val := binary.LittleEndian.Uint64(data[pos : pos+8])
				if val > math.MaxInt64 {
					dest[i] = uint64ToString(val)
				} else {
					dest[i] = int64(val)
				}
			} else {
				dest[i] = int64(binary.LittleEndian.Uint64(data[pos : pos+8]))
			}
			pos += 8
			continue

		case fieldTypeFloat:
			dest[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
			pos += 4
			continue

		case fieldTypeDouble:
			dest[i] = math.Float64frombits(binary.LittleEndian.Uint64(data[pos : pos+8]))
			pos += 8
			continue

		// Length coded Binary Strings
		case fieldTypeDecimal, fieldTypeNewDecimal, fieldTypeVarChar,
			fieldTypeBit, fieldTypeEnum, fieldTypeSet, fieldTypeTinyBLOB,
			fieldTypeMediumBLOB, fieldTypeLongBLOB, fieldTypeBLOB,
			fieldTypeVarString, fieldTypeString, fieldTypeGeometry:
			var isNull bool
			var n int
			dest[i], isNull, n, err = readLengthEnodedString(data[pos:])
			pos += n
			if err == nil {
				if !isNull {
					continue
				} else {
					dest[i] = nil
					continue
				}
			}
			return err

		// Date YYYY-MM-DD
		case fieldTypeDate, fieldTypeNewDate:
			num, isNull, n := readLengthEncodedInteger(data[pos:])
			pos += n

			if isNull {
				dest[i] = nil
				continue
			}

			if rows.mc.parseTime {
				dest[i], err = parseBinaryDateTime(num, data[pos:], rows.mc.cfg.loc)
			} else {
				dest[i], err = formatBinaryDate(num, data[pos:])
			}

			if err == nil {
				pos += int(num)
				continue
			} else {
				return err
			}

		// Time [-][H]HH:MM:SS[.fractal]
		case fieldTypeTime:
			num, isNull, n := readLengthEncodedInteger(data[pos:])
			pos += n

			if num == 0 {
				if isNull {
					dest[i] = nil
					continue
				} else {
					dest[i] = []byte("00:00:00")
					continue
				}
			}

			var sign string
			if data[pos] == 1 {
				sign = "-"
			}

			switch num {
			case 8:
				dest[i] = []byte(fmt.Sprintf(
					sign+"%02d:%02d:%02d",
					uint16(data[pos+1])*24+uint16(data[pos+5]),
					data[pos+6],
					data[pos+7],
				))
				pos += 8
				continue
			case 12:
				dest[i] = []byte(fmt.Sprintf(
					sign+"%02d:%02d:%02d.%06d",
					uint16(data[pos+1])*24+uint16(data[pos+5]),
					data[pos+6],
					data[pos+7],
					binary.LittleEndian.Uint32(data[pos+8:pos+12]),
				))
				pos += 12
				continue
			default:
				return fmt.Errorf("Invalid TIME-packet length %d", num)
			}

		// Timestamp YYYY-MM-DD HH:MM:SS[.fractal]
		case fieldTypeTimestamp, fieldTypeDateTime:
			num, isNull, n := readLengthEncodedInteger(data[pos:])

			pos += n

			if isNull {
				dest[i] = nil
				continue
			}

			if rows.mc.parseTime {
				dest[i], err = parseBinaryDateTime(num, data[pos:], rows.mc.cfg.loc)
			} else {
				dest[i], err = formatBinaryDateTime(num, data[pos:])
			}

			if err == nil {
				pos += int(num)
				continue
			} else {
				return err
			}

		// Please report if this happens!
		default:
			return fmt.Errorf("Unknown FieldType %d", rows.columns[i].fieldType)
		}
	}

	return nil
}
