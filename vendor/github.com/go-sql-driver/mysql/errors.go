// Go MySQL Driver - A MySQL-Driver for Go's database/sql package
//
// Copyright 2013 The Go-MySQL-Driver Authors. All rights reserved.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

package mysql

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
)

var (
	errInvalidConn = errors.New("Invalid Connection")
	errMalformPkt  = errors.New("Malformed Packet")
	errNoTLS       = errors.New("TLS encryption requested but server does not support TLS")
	errOldPassword = errors.New("This server only supports the insecure old password authentication. If you still want to use it, please add 'allowOldPasswords=1' to your DSN. See also https://github.com/go-sql-driver/mysql/wiki/old_passwords")
	errOldProtocol = errors.New("MySQL-Server does not support required Protocol 41+")
	errPktSync     = errors.New("Commands out of sync. You can't run this command now")
	errPktSyncMul  = errors.New("Commands out of sync. Did you run multiple statements at once?")
	errPktTooLarge = errors.New("Packet for query is too large. You can change this value on the server by adjusting the 'max_allowed_packet' variable.")
)

// error type which represents a single MySQL error
type MySQLError struct {
	Number  uint16
	Message string
}

func (me *MySQLError) Error() string {
	return fmt.Sprintf("Error %d: %s", me.Number, me.Message)
}

// error type which represents a group of one or more MySQL warnings
type MySQLWarnings []mysqlWarning

func (mws MySQLWarnings) Error() string {
	var msg string
	for i, warning := range mws {
		if i > 0 {
			msg += "\r\n"
		}
		msg += fmt.Sprintf("%s %s: %s", warning.Level, warning.Code, warning.Message)
	}
	return msg
}

// error type which represents a single MySQL warning
type mysqlWarning struct {
	Level   string
	Code    string
	Message string
}

func (mc *mysqlConn) getWarnings() (err error) {
	rows, err := mc.Query("SHOW WARNINGS", []driver.Value{})
	if err != nil {
		return
	}

	var warnings = MySQLWarnings{}
	var values = make([]driver.Value, 3)

	var warning mysqlWarning
	var raw []byte
	var ok bool

	for {
		err = rows.Next(values)
		switch err {
		case nil:
			warning = mysqlWarning{}

			if raw, ok = values[0].([]byte); ok {
				warning.Level = string(raw)
			} else {
				warning.Level = fmt.Sprintf("%s", values[0])
			}
			if raw, ok = values[1].([]byte); ok {
				warning.Code = string(raw)
			} else {
				warning.Code = fmt.Sprintf("%s", values[1])
			}
			if raw, ok = values[2].([]byte); ok {
				warning.Message = string(raw)
			} else {
				warning.Message = fmt.Sprintf("%s", values[0])
			}

			warnings = append(warnings, warning)

		case io.EOF:
			return warnings

		default:
			rows.Close()
			return
		}
	}
}
