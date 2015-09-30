// Go MySQL Driver - A MySQL-Driver for Go's database/sql package
//
// Copyright 2013 The Go-MySQL-Driver Authors. All rights reserved.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

package mysql

import (
	"fmt"
	"testing"
	"time"
)

var testDSNs = []struct {
	in  string
	out string
	loc *time.Location
}{
	{"username:password@protocol(address)/dbname?param=value", "&{user:username passwd:password net:protocol addr:address dbname:dbname params:map[param:value] loc:%p timeout:0 tls:<nil> allowAllFiles:false allowOldPasswords:false clientFoundRows:false}", time.UTC},
	{"user@unix(/path/to/socket)/dbname?charset=utf8", "&{user:user passwd: net:unix addr:/path/to/socket dbname:dbname params:map[charset:utf8] loc:%p timeout:0 tls:<nil> allowAllFiles:false allowOldPasswords:false clientFoundRows:false}", time.UTC},
	{"user:password@tcp(localhost:5555)/dbname?charset=utf8&tls=true", "&{user:user passwd:password net:tcp addr:localhost:5555 dbname:dbname params:map[charset:utf8] loc:%p timeout:0 tls:<nil> allowAllFiles:false allowOldPasswords:false clientFoundRows:false}", time.UTC},
	{"user:password@tcp(localhost:5555)/dbname?charset=utf8mb4,utf8&tls=skip-verify", "&{user:user passwd:password net:tcp addr:localhost:5555 dbname:dbname params:map[charset:utf8mb4,utf8] loc:%p timeout:0 tls:<nil> allowAllFiles:false allowOldPasswords:false clientFoundRows:false}", time.UTC},
	{"user:password@/dbname?loc=UTC&timeout=30s&allowAllFiles=1&clientFoundRows=true&allowOldPasswords=TRUE", "&{user:user passwd:password net:tcp addr:127.0.0.1:3306 dbname:dbname params:map[] loc:%p timeout:30000000000 tls:<nil> allowAllFiles:true allowOldPasswords:true clientFoundRows:true}", time.UTC},
	{"user:p@ss(word)@tcp([de:ad:be:ef::ca:fe]:80)/dbname?loc=Local", "&{user:user passwd:p@ss(word) net:tcp addr:[de:ad:be:ef::ca:fe]:80 dbname:dbname params:map[] loc:%p timeout:0 tls:<nil> allowAllFiles:false allowOldPasswords:false clientFoundRows:false}", time.Local},
	{"/dbname", "&{user: passwd: net:tcp addr:127.0.0.1:3306 dbname:dbname params:map[] loc:%p timeout:0 tls:<nil> allowAllFiles:false allowOldPasswords:false clientFoundRows:false}", time.UTC},
	{"@/", "&{user: passwd: net:tcp addr:127.0.0.1:3306 dbname: params:map[] loc:%p timeout:0 tls:<nil> allowAllFiles:false allowOldPasswords:false clientFoundRows:false}", time.UTC},
	{"/", "&{user: passwd: net:tcp addr:127.0.0.1:3306 dbname: params:map[] loc:%p timeout:0 tls:<nil> allowAllFiles:false allowOldPasswords:false clientFoundRows:false}", time.UTC},
	{"", "&{user: passwd: net:tcp addr:127.0.0.1:3306 dbname: params:map[] loc:%p timeout:0 tls:<nil> allowAllFiles:false allowOldPasswords:false clientFoundRows:false}", time.UTC},
	{"user:p@/ssword@/", "&{user:user passwd:p@/ssword net:tcp addr:127.0.0.1:3306 dbname: params:map[] loc:%p timeout:0 tls:<nil> allowAllFiles:false allowOldPasswords:false clientFoundRows:false}", time.UTC},
	{"unix/?arg=%2Fsome%2Fpath.ext", "&{user: passwd: net:unix addr:/tmp/mysql.sock dbname: params:map[arg:/some/path.ext] loc:%p timeout:0 tls:<nil> allowAllFiles:false allowOldPasswords:false clientFoundRows:false}", time.UTC},
}

func TestDSNParser(t *testing.T) {
	var cfg *config
	var err error
	var res string

	for i, tst := range testDSNs {
		cfg, err = parseDSN(tst.in)
		if err != nil {
			t.Error(err.Error())
		}

		// pointer not static
		cfg.tls = nil

		res = fmt.Sprintf("%+v", cfg)
		if res != fmt.Sprintf(tst.out, tst.loc) {
			t.Errorf("%d. parseDSN(%q) => %q, want %q", i, tst.in, res, fmt.Sprintf(tst.out, tst.loc))
		}
	}
}

func TestDSNParserInvalid(t *testing.T) {
	var invalidDSNs = []string{
		"@net(addr/",  // no closing brace
		"@tcp(/",      // no closing brace
		"tcp(/",       // no closing brace
		"(/",          // no closing brace
		"net(addr)//", // unescaped
		//"/dbname?arg=/some/unescaped/path",
	}

	for i, tst := range invalidDSNs {
		if _, err := parseDSN(tst); err == nil {
			t.Errorf("invalid DSN #%d. (%s) didn't error!", i, tst)
		}
	}
}

func BenchmarkParseDSN(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, tst := range testDSNs {
			if _, err := parseDSN(tst.in); err != nil {
				b.Error(err.Error())
			}
		}
	}
}

func TestScanNullTime(t *testing.T) {
	var scanTests = []struct {
		in    interface{}
		error bool
		valid bool
		time  time.Time
	}{
		{tDate, false, true, tDate},
		{sDate, false, true, tDate},
		{[]byte(sDate), false, true, tDate},
		{tDateTime, false, true, tDateTime},
		{sDateTime, false, true, tDateTime},
		{[]byte(sDateTime), false, true, tDateTime},
		{tDate0, false, true, tDate0},
		{sDate0, false, true, tDate0},
		{[]byte(sDate0), false, true, tDate0},
		{sDateTime0, false, true, tDate0},
		{[]byte(sDateTime0), false, true, tDate0},
		{"", true, false, tDate0},
		{"1234", true, false, tDate0},
		{0, true, false, tDate0},
	}

	var nt = NullTime{}
	var err error

	for _, tst := range scanTests {
		err = nt.Scan(tst.in)
		if (err != nil) != tst.error {
			t.Errorf("%v: expected error status %t, got %t", tst.in, tst.error, (err != nil))
		}
		if nt.Valid != tst.valid {
			t.Errorf("%v: expected valid status %t, got %t", tst.in, tst.valid, nt.Valid)
		}
		if nt.Time != tst.time {
			t.Errorf("%v: expected time %v, got %v", tst.in, tst.time, nt.Time)
		}
	}
}
