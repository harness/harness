// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package token

import (
	"strconv"
	"testing"
	"time"

	"github.com/bradrydzewski/my-app/types"

	"github.com/dgrijalva/jwt-go"
)

func TestToken(t *testing.T) {
	user := &types.User{ID: 42, Admin: true}
	tokenStr, err := Generate(user, "TEST0E4C2F76C58916E")
	if err != nil {
		t.Error(err)
		return
	}

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		sub := token.Claims.(*Claims).Subject
		id, _ := strconv.ParseInt(sub, 10, 64)
		if id != 42 {
			t.Errorf("want subscriber id, got %v", id)
		}
		return []byte("TEST0E4C2F76C58916E"), nil
	})
	if err != nil {
		t.Error(err)
		return
	}
	if token.Valid == false {
		t.Errorf("invalid token")
		return
	}
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		t.Errorf("invalid token signing method")
		return
	}

	if expires := token.Claims.(*Claims).ExpiresAt; expires > 0 {
		if time.Now().Unix() > expires {
			t.Errorf("token expired")
		}
	}
}

func TestTokenExpired(t *testing.T) {
	user := &types.User{ID: 42, Admin: true}
	tokenStr, err := GenerateExp(user, 1637549186, "TEST0E4C2F76C58916E")
	if err != nil {
		t.Error(err)
		return
	}

	_, err = jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		sub := token.Claims.(*Claims).Subject
		id, _ := strconv.ParseInt(sub, 10, 64)
		if id != 42 {
			t.Errorf("want subscriber id, got %v", id)
		}
		return []byte("TEST0E4C2F76C58916E"), nil
	})
	if err == nil {
		t.Errorf("expect token expired")
		return
	}
}

func TestTokenNotExpired(t *testing.T) {
	user := &types.User{ID: 42, Admin: true}
	tokenStr, err := GenerateExp(user, time.Now().Add(time.Hour).Unix(), "TEST0E4C2F76C58916E")
	if err != nil {
		t.Error(err)
		return
	}

	token_, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		sub := token.Claims.(*Claims).Subject
		id, _ := strconv.ParseInt(sub, 10, 64)
		if id != 42 {
			t.Errorf("want subscriber id, got %v", id)
		}
		return []byte("TEST0E4C2F76C58916E"), nil
	})
	if err != nil {
		t.Error(err)
		return
	}

	if claims, ok := token_.Claims.(*Claims); ok {
		if claims.ExpiresAt > 0 {
			if time.Now().Unix() > claims.ExpiresAt {
				t.Errorf("expect token not expired")
			}
		} else {
			t.Errorf("expect token expiration greater than zero")
		}
	} else {
		t.Errorf("expect token claims from token")
	}
}
