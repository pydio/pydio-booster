// Package pydhttp contains all http related work
/*
 * Copyright 2007-2016 Abstrium <contact (at) pydio.com>
 * This file is part of Pydio.
 *
 * Pydio is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Pydio is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Pydio.  If not, see <http://www.gnu.org/licenses/>.
 *
 * The latest code can be found at <https://pydio.com/>.
 */
package pydhttp

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

// Token public and private parts
type Token struct {
	T string
	P string
}

// NewToken for the API
func NewToken(token string, password string) *Token {
	return &Token{
		T: token,
		P: password,
	}

}

// NewTokenFromURLWithCookie retrieves a token pair by sending a message containing a cookie to a specific URL
func NewTokenFromURLWithCookie(url *url.URL, cookie *http.Cookie) (token *Token, err error) {
	client := NewClient()

	req, err := http.NewRequest("GET", url.String(), nil)
	req.AddCookie(cookie)

	resp, err := client.Do(req)

	if err != nil {
		return nil, errors.New("NewTokenFromURLWithCookie: Could not retrieve token")
	}

	defer resp.Body.Close()

	token = NewToken("", "")

	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(token); err != nil {
		return nil, errors.New("NewTokenFromURLWithCookie: Could not decrypt token ")
	}

	return token, err
}

// NewTokenFromURLWithBasicAuth retrieves a token pair by sending a message with query arguments
func NewTokenFromURLWithBasicAuth(url *url.URL, username string, password string) (token *Token, err error) {
	client := NewClient()

	req, err := http.NewRequest("GET", url.String(), nil)
	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)

	if err != nil {
		return nil, errors.New("NewTokenFromURLWithBasicAuth: Could not retrieve token")
	}

	defer resp.Body.Close()

	token = NewToken("", "")

	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(token); err != nil {
		return nil, errors.New("NewTokenFromURLWithBasicAuth: Could not decrypt token")
	}

	return token, err
}

// NewTokenFromJWT creates a token based on a JWT token string
func NewTokenFromJWT(str string, signatureSecret string) (token *Token, err error) {

	decryptedToken, err := jwt.Parse(str, func(token *jwt.Token) (interface{}, error) {
		return []byte(signatureSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !decryptedToken.Valid {
		return nil, errors.New("Token has been tampered with")
	}

	if err != nil {
		return nil, err
	}

	token = NewToken("", "")

	claim := decryptedToken.Claims["token"]
	claimStr, ok := claim.(string)

	if !ok {
		return nil, errors.New("Token format is not valid")
	}

	b, err := jwt.DecodeSegment(claimStr)

	// Unmarshalling to an Token Structure
	err = json.Unmarshal(b, &token)

	return token, nil
}

// GetQueryArgs for the token
func (t *Token) GetQueryArgs(uri string) *Auth {

	if t == nil {
		return nil
	}

	replacer := strings.NewReplacer("%2F", "/")
	uri = replacer.Replace(url.QueryEscape(uri))

	b := randomString(10)

	sha1 := sha1.New()
	sha1.Write([]byte(b))

	nonce := hex.EncodeToString(sha1.Sum(nil))
	uri = strings.TrimRight(uri, "/")
	msg := string(uri[:]) + ":" + string(nonce[:]) + ":" + t.P
	hmac := hmac.New(sha256.New, []byte(t.T))
	hmac.Write([]byte(msg))
	hmacMsg := hex.EncodeToString(hmac.Sum(nil))

	hash := nonce + ":" + hmacMsg

	return &Auth{
		Token: t.T,
		Hash:  hash,
	}
}

// JWT representation of the Token
func (t *Token) JWT(signatureSecret string, hoursBeforeExpiry int) (string, error) {
	payload, err := json.Marshal(t)
	if err != nil {
		return "", err
	}

	// Create the token
	token := jwt.New(jwt.SigningMethodHS256)

	// Set some claims
	token.Claims["token"] = payload

	token.Claims["exp"] = time.Now().Add(time.Hour * time.Duration(hoursBeforeExpiry)).Unix()

	// Sign and get the complete encoded token as a string
	str, err := token.SignedString([]byte(signatureSecret))
	if err != nil {
		return "", err
	}

	return str, nil

}
