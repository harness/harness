// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package token

import (
	"testing"
)

func TestToken(t *testing.T) {
	// user := &types.User{ID: 42, Admin: true}
	// tokenStr, err := GenerateJWT(user, "TEST0E4C2F76C58916E")
	// if err != nil {
	// 	t.Error(err)
	// 	return
	// }

	// token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
	// 	sub := token.Claims.(*JWTClaims).Subject
	// 	id, _ := strconv.ParseInt(sub, 10, 64)
	// 	if id != 42 {
	// 		t.Errorf("want subscriber id, got %v", id)
	// 	}
	// 	return []byte("TEST0E4C2F76C58916E"), nil
	// })
	// if err != nil {
	// 	t.Error(err)
	// 	return
	// }
	// if token.Valid == false {
	// 	t.Errorf("invalid token")
	// 	return
	// }
	// if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
	// 	t.Errorf("invalid token signing method")
	// 	return
	// }

	// if expires := token.Claims.(*JWTClaims).ExpiresAt; expires > 0 {
	// 	if time.Now().Unix() > expires {
	// 		t.Errorf("token expired")
	// 	}
	// }
}

func TestTokenExpired(t *testing.T) {
	// user := &types.User{ID: 42, Admin: true}
	// tokenStr, err := GenerateJWTWithExpiration(user, 1637549186, "TEST0E4C2F76C58916E")
	// if err != nil {
	// 	t.Error(err)
	// 	return
	// }

	// _, err = jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
	// 	sub := token.Claims.(*JWTClaims).Subject
	// 	id, _ := strconv.ParseInt(sub, 10, 64)
	// 	if id != 42 {
	// 		t.Errorf("want subscriber id, got %v", id)
	// 	}
	// 	return []byte("TEST0E4C2F76C58916E"), nil
	// })
	// if err == nil {
	// 	t.Errorf("expect token expired")
	// 	return
	// }
}

func TestTokenNotExpired(t *testing.T) {
	// user := &types.User{ID: 42, Admin: true}
	// tokenStr, err := GenerateJWTWithExpiration(user, time.Now().Add(time.Hour).Unix(), "TEST0E4C2F76C58916E")
	// if err != nil {
	// 	t.Error(err)
	// 	return
	// }

	// token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
	// 	sub := token.Claims.(*JWTClaims).Subject
	// 	id, _ := strconv.ParseInt(sub, 10, 64)
	// 	if id != 42 {
	// 		t.Errorf("want subscriber id, got %v", id)
	// 	}
	// 	return []byte("TEST0E4C2F76C58916E"), nil
	// })
	// if err != nil {
	// 	t.Error(err)
	// 	return
	// }

	// claims, ok := token.Claims.(*JWTClaims)
	// if !ok {
	// 	t.Errorf("expect token claims from token")
	// 	return
	// }

	// if claims.ExpiresAt > 0 && time.Now().Unix() > claims.ExpiresAt {
	// 	t.Errorf("expect token not expired")
	// 	return
	// }
}
