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
	"encoding/json"
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// User format definition
type User struct {
	ID        string `xml:"id,attr" json:"id"`
	GroupPath string `xml:"groupPath,attr" json:"grp"`
	Repos     []Repo `xml:"repositories>repo" json:"rep"`
}

// NewUser object with pipewriting
func NewUser() *User {
	return &User{}
}

// NewUserFromJWT creates a user based on a JWT token string
func NewUserFromJWT(token string, signatureSecret string) (*User, error) {
	decryptedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(signatureSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !decryptedToken.Valid {
		return nil, errors.New("Token has been tampered with")
	}

	claim := decryptedToken.Claims["user"]
	claimStr, ok := claim.(string)

	if !ok {
		return nil, errors.New("User format is not valid")
	}

	userStr, err := jwt.DecodeSegment(claimStr)
	if err != nil {
		return nil, err
	}

	var user User

	err = json.Unmarshal([]byte(userStr), &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetRepo via its ID
func (u *User) GetRepo(id string) *Repo {
	for _, repo := range u.Repos {
		if repo.ID == id {
			return &repo
		}
	}

	return nil
}

// JWT representation of the User
func (u *User) JWT(signatureSecret string, hoursBeforeExpiry int) (string, error) {
	payload, err := json.Marshal(u)
	if err != nil {
		return "", err
	}

	// Create the token
	token := jwt.New(jwt.SigningMethodHS256)
	// Set some claims
	token.Claims["user"] = payload

	token.Claims["exp"] = time.Now().Add(time.Hour * time.Duration(hoursBeforeExpiry)).Unix()

	// Sign and get the complete encoded token as a string
	tokenString, err := token.SignedString([]byte(signatureSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// Read the node by encoding to its json representation
func (u *User) Read(p []byte) (int, error) {
	data, err := json.Marshal(u)

	numBytes := copy(p, data)

	return numBytes, err
}
