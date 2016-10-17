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
	"errors"
	"math/rand"
	"net/http"
	"net/url"
	"time"
)

// API data
type API struct {
	tokenURL url.URL
	token    *Token
}

// Auth data format for a query
type Auth struct {
	Token string
	Hash  string
	Key   string
}

// NewAPI reference
func NewAPI(url url.URL, auth ...interface{}) (api *API, err error) {

	var token *Token

	api = &API{
		tokenURL: url,
	}

	if len(auth) == 1 {
		if cookie, ok := auth[0].(*http.Cookie); ok {
			token, err = NewTokenFromURLWithCookie(&url, cookie)

			if err != nil {
				return nil, err
			}
		}
	} else if len(auth) == 2 {

		if username, ok := auth[0].(string); ok {
			if password, ok := auth[1].(string); ok {
				token, err = NewTokenFromURLWithBasicAuth(&url, username, password)

				if err != nil {
					return nil, err
				}
			}
		}
	}

	api.token = token

	return api, nil
}

// GetQueryArgs based on the uri given for the API auth
func (api *API) GetQueryArgs(uri string) *Auth {

	if api == nil {
		return nil
	}

	return api.token.GetQueryArgs(uri)

}

// GetBaseURL returns the api base url
func (api *API) GetBaseURL() (*url.URL, error) {

	if api == nil {
		return nil, errors.New("API is empty")
	}

	rootURL, err := url.Parse("/")
	if err != nil {
		return nil, err
	}

	baseURL := api.tokenURL.ResolveReference(rootURL)
	return baseURL, nil
}

func randomString(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}
