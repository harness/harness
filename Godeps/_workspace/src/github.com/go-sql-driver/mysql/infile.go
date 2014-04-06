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
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	fileRegister   map[string]bool
	readerRegister map[string]func() io.Reader
)

func init() {
	fileRegister = make(map[string]bool)
	readerRegister = make(map[string]func() io.Reader)
}

// RegisterLocalFile adds the given file to the file whitelist,
// so that it can be used by "LOAD DATA LOCAL INFILE <filepath>".
// Alternatively you can allow the use of all local files with
// the DSN parameter 'allowAllFiles=true'
//
//  filePath := "/home/gopher/data.csv"
//  mysql.RegisterLocalFile(filePath)
//  err := db.Exec("LOAD DATA LOCAL INFILE '" + filePath + "' INTO TABLE foo")
//  if err != nil {
//  ...
//
func RegisterLocalFile(filePath string) {
	fileRegister[strings.Trim(filePath, `"`)] = true
}

// DeregisterLocalFile removes the given filepath from the whitelist.
func DeregisterLocalFile(filePath string) {
	delete(fileRegister, strings.Trim(filePath, `"`))
}

// RegisterReaderHandler registers a handler function which is used
// to receive a io.Reader.
// The Reader can be used by "LOAD DATA LOCAL INFILE Reader::<name>".
// If the handler returns a io.ReadCloser Close() is called when the
// request is finished.
//
//  mysql.RegisterReaderHandler("data", func() io.Reader {
//  	var csvReader io.Reader // Some Reader that returns CSV data
//  	... // Open Reader here
//  	return csvReader
//  })
//  err := db.Exec("LOAD DATA LOCAL INFILE 'Reader::data' INTO TABLE foo")
//  if err != nil {
//  ...
//
func RegisterReaderHandler(name string, handler func() io.Reader) {
	readerRegister[name] = handler
}

// DeregisterReaderHandler removes the ReaderHandler function with
// the given name from the registry.
func DeregisterReaderHandler(name string) {
	delete(readerRegister, name)
}

func (mc *mysqlConn) handleInFileRequest(name string) (err error) {
	var rdr io.Reader
	data := make([]byte, 4+mc.maxWriteSize)

	if strings.HasPrefix(name, "Reader::") { // io.Reader
		name = name[8:]
		handler, inMap := readerRegister[name]
		if handler != nil {
			rdr = handler()
		}
		if rdr == nil {
			if !inMap {
				err = fmt.Errorf("Reader '%s' is not registered", name)
			} else {
				err = fmt.Errorf("Reader '%s' is <nil>", name)
			}
		}
	} else { // File
		name = strings.Trim(name, `"`)
		if mc.cfg.allowAllFiles || fileRegister[name] {
			rdr, err = os.Open(name)
		} else {
			err = fmt.Errorf("Local File '%s' is not registered. Use the DSN parameter 'allowAllFiles=true' to allow all files", name)
		}
	}

	if rdc, ok := rdr.(io.ReadCloser); ok {
		defer func() {
			if err == nil {
				err = rdc.Close()
			} else {
				rdc.Close()
			}
		}()
	}

	// send content packets
	var ioErr error
	if err == nil {
		var n int
		for err == nil && ioErr == nil {
			n, err = rdr.Read(data[4:])
			if n > 0 {
				data[0] = byte(n)
				data[1] = byte(n >> 8)
				data[2] = byte(n >> 16)
				data[3] = mc.sequence
				ioErr = mc.writePacket(data[:4+n])
			}
		}
		if err == io.EOF {
			err = nil
		}
		if ioErr != nil {
			errLog.Print(ioErr.Error())
			return driver.ErrBadConn
		}
	}

	// send empty packet (termination)
	ioErr = mc.writePacket([]byte{
		0x00,
		0x00,
		0x00,
		mc.sequence,
	})
	if ioErr != nil {
		errLog.Print(ioErr.Error())
		return driver.ErrBadConn
	}

	// read OK packet
	if err == nil {
		return mc.readResultOK()
	} else {
		mc.readPacket()
	}
	return err
}
