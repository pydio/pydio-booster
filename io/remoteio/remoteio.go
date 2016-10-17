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
package remoteio

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strings"

	pydhttp "github.com/pydio/pydio-booster/http"
	pydio "github.com/pydio/pydio-booster/io"
)

// Arg format
type Arg struct {
	key   string
	value string
}

// Write for the moment is handled by the PHP
// so sending a request there
func Write(api *pydhttp.API) func(io.Reader, pydio.Node) (int64, error) {
	return func(r io.Reader, n pydio.Node) (int64, error) {
		uri := fmt.Sprintf("/api/%s/upload/put", n.Repo.ID)
		auth := api.GetQueryArgs(uri)

		// Set up a pipe from which the body can be read.
		pr, pw := io.Pipe()

		w := multipart.NewWriter(pw)

		// Close the reader if we exit early
		defer pr.Close()

		args := []Arg{
			Arg{key: "force_post", value: "true"},
			Arg{key: "auth_hash", value: auth.Hash},
			Arg{key: "auth_token", value: auth.Token},
			Arg{key: "tmp_repository_id", value: n.Repo.ID},
			Arg{key: "dir", value: n.Dir.String()},
			Arg{key: "urlencoded_filename", value: n.Basename},
			Arg{key: "xhr_uploader", value: "true"},
			Arg{key: "XDEBUG_SESSION_START", value: "phpstorm"},
		}

		// Write the request body based on the request
		done := WriteBody(w, r, args)

		numBytes := int64(0)
		go func() {
			defer pw.Close()
			numBytes = <-done
		}()

		// Create the HTTP request.
		apiURL, _ := api.GetBaseURL()
		req, _ := http.NewRequest("POST", strings.TrimRight(apiURL.String(), "/")+uri, pr)
		req.Header.Add("Content-Type", w.FormDataContentType())
		req.Body = ioutil.NopCloser(pr)

		// Execute the HTTP request.
		client := &http.Client{
			CheckRedirect: func(r *http.Request, via []*http.Request) error {
				return nil
			},
		}
		resp, err := client.Do(req)
		if err != nil {
			return 0, err
		}

		if resp.StatusCode != http.StatusOK {
			return 0, errors.New("Failed to upload data: " + resp.Status)
		}

		return numBytes, err
	}
}

// WriteBody of the request to a multipart writer
func WriteBody(w *multipart.Writer, r io.Reader, args []Arg) chan (int64) {

	c := make(chan (int64))

	go func() {
		var err error

		// Add Arguments to the request
		for _, arg := range args {
			w.WriteField(arg.key, arg.value)
		}

		fw, _ := w.CreateFormFile("userfile_0", "userfile_0")

		// Write the body into the multipart writer.
		numBytes, err := io.Copy(fw, r)
		if err != nil {
			return
		}

		// Finish the multipart body.
		err = w.Close()
		if err != nil {
			return
		}

		c <- numBytes
	}()

	return c
}
