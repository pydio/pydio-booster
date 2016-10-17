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
package pydio

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

// Options format definition
type Options struct {
	PartialTargetBytesize  int64  `json:"partial_target_bytesize"`
	PartialUpload          bool   `json:"partial_upload"`
	XHRUploader            bool   `json:"xhr_uploader"`
	ForcePost              bool   `json:"force_post"`
	URLEncodedFilename     string `json:"urlencoded_filename"`
	AppendToURLEncodedPart string `json:"appendto_urlencoded_part"`
	Path                   string `json:"PATH"`

	FileOptions `json:"OPTIONS"`
	S3Options
}

// FileOptions format definition
type FileOptions struct {
	Type string `json:"TYPE"`
	Path string `json:"PATH"`
}

// S3Options from server
type S3Options struct {
	Type              string `json:"TYPE"`
	APIKey            string `json:"API_KEY"`
	APIVersion        string `json:"API_VERSION"`
	Container         string `json:"CONTAINER"`
	Proxy             string `json:"PROXY"`
	VHostNotSupported bool   `json:"VHOST_NOT_SUPPORTED"`
	Region            string `json:"REGION"`
	SecretKey         string `json:"SECRET_KEY"`
	SignatureVersion  string `json:"SIGNATURE_VERSION"`
	StorageURL        string `json:"STORAGE_URL"`
}

// NewOptionsFromJWT creates a user based on a JWT token string
func NewOptionsFromJWT(token string, signatureSecret string) (*Options, error) {
	decryptedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
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

	var options *Options

	log.Println(decryptedToken.Claims)

	claim := decryptedToken.Claims["options"]
	claimStr, ok := claim.(string)

	if !ok {
		return nil, errors.New("Options format is not valid")
	}

	b, err := jwt.DecodeSegment(claimStr)

	// Unmarshalling to an Options Structure
	err = json.Unmarshal(b, &options)

	return options, nil
}

// JWT representation of the User
func (o *Options) JWT(signatureSecret string, hoursBeforeExpiry int) (string, error) {
	payload, err := json.Marshal(o)
	if err != nil {
		return "", err
	}

	// Create the token
	token := jwt.New(jwt.SigningMethodHS256)
	// Set some claims
	token.Claims["options"] = payload

	token.Claims["exp"] = time.Now().Add(time.Hour * time.Duration(hoursBeforeExpiry)).Unix()

	// Sign and get the complete encoded token as a string
	tokenString, err := token.SignedString([]byte(signatureSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// Read the options by encoding to its json representation
func (o *Options) Read(p []byte) (int, error) {

	buf := bytes.NewBuffer(p)
	enc := json.NewEncoder(buf)

	enc.Encode(o)

	return buf.Len(), io.EOF
}
