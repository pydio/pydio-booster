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
	"crypto/tls"
	"errors"
	"log"
	"net/http"
)

var (
	// ErrRedirectViaPydioClient is the error used on CheckRedirect
	ErrRedirectViaPydioClient = errors.New("Redirection is handled by the Pydio HTTP Client")
)

// Client extension to the http client
type Client http.Client

// NewClient with Redirection handling
func NewClient() *Client {

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return &Client{
		Transport: tr,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			return ErrRedirectViaPydioClient
		},
	}
}

// Do the Request through Client
func (c *Client) Do(req *http.Request) (resp *http.Response, err error) {
	resp, err = (*http.Client)(c).Do(req)

	if err != nil {
		log.Println("Error while sending request ", err)
		return
	}

	if shouldRedirectGet(resp.StatusCode) {
		if loc, err := resp.Location(); err == nil {
			log.Println("Redirecting to ", loc)
			req.URL = loc
			return c.Do(req)
		}
	}
	return
}

func shouldRedirectGet(statusCode int) bool {
	switch statusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect:
		return true
	}
	return false
}
