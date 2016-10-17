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
package pydiomiddleware

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/mholt/caddy/caddyhttp/httpserver"
	"github.com/pydio/pydio-booster/http"
	"github.com/pydio/pydio-booster/worker"
	"github.com/pydio/go/http"
)

var client = pydhttp.NewClient()

// RequestJob definition for the uploader
type RequestJob struct {
	Request    http.Request
	HandleFunc func(io.Reader) error
	ErrorFunc  func()
}

// Do the job
func (j *RequestJob) Do() (err error) {

	resp, err := client.Do(&j.Request)
	if err != nil {
		j.ErrorFunc()
		return
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if err != nil && err != io.EOF {
		j.ErrorFunc()
		return
	}

	if resp.StatusCode != http.StatusOK || err != nil {
		log.Printf("[ERROR] Not authorized : %v %v", j.Request, resp)
		j.ErrorFunc()
		return err
	}

	return j.HandleFunc(resp.Body)
}

// NewRequestJob prepares the job for the middleware request
// based on the rules
func NewRequestJob(
	u url.URL,
	headers [][2]string,
	cookies []*http.Cookie,
	out string,
	ctx context.Context,
	replacer httpserver.Replacer,
	encoder json.Encoder,
	writer io.Writer,
	close func() error,
	cancel func(),
) (pydioworker.Job, error) {

	queryArgs := u.Query()
	log.Println("[REQ:MW] Request Job Start", u, headers, queryArgs, out)

	node, err := getContextNode(ctx)
	if err == nil {
		log.Println("[INFO:MW] Request : Retrieved node ", node.Repo)
		var repo *url.URL
		var dir *url.URL

		repo = &url.URL{Path: node.Repo.String()}
		dir = &url.URL{Path: strings.TrimPrefix(node.Dir.String(), "/")}

		replacer.Set("repo", repo.String())
		replacer.Set("nodedir", dir.String())
		replacer.Set("nodename", node.Basename)

		log.Println("[INFO:MW] Request : Replacer is set ", replacer)
	} else {
		log.Println("[ERROR:MW] Request : Could not read node")
	}

	u.Path = replacer.Replace(u.Path)

	values := u.Query()
	if out != "token" {
		if auth, errAuth := getContextAuthParams(ctx, u.Path); errAuth == nil {
			values.Set("auth_hash", auth.Hash)
			values.Set("auth_token", auth.Token)
			if auth.Key != "" {
				values.Set("key", auth.Key)
			}
		}
	}
	u.RawQuery = values.Encode()

	request, _ := http.NewRequest("GET", u.String(), nil)
	log.Println("[DEBUG:MW] Doing headers")
	for _, header := range headers {
		request.Header.Add(header[0], replacer.Replace(header[1]))
	}

	log.Println("[DEBUG:MW] Doing cookies")
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}

	request.URL = &u

	log.Printf("[DEBUG:MW] URL is %s - headers - %v cookies - %v", u.String(), headers, request.Cookies())

	job := &RequestJob{
		Request:   *request,
		ErrorFunc: cancel,
		HandleFunc: func(r io.Reader) error {
			defer close()

			var q query
			var dec decoder

			log.Println("Out to ", out)

			switch out {
			case "body":
				// Write response header
				//writeHeader(w, resp)

				// Write the response body
				//data, _ := ioutil.ReadAll(r)
				//log.Printf("%s", data)

				_, err = io.Copy(writer, r)
				if err != nil {
					log.Printf("[ERROR:MW] Could not write body %v", err)
					return err
				}

				return nil

			case "user":
				q = &UserQuery{}
				dec = xml.NewDecoder(r)

			case "options":
				q = &OptionsQuery{}
				dec = json.NewDecoder(r)

			case "proxy":
				q = &ProxyQuery{}
				dec = json.NewDecoder(r)

			case "token":
				q = &TokenQuery{}
				dec = json.NewDecoder(r)

			case "client":
				q = &ClientQuery{}
				dec = json.NewDecoder(r)

			default:
				return errors.New("[ERROR:MW] Wrong decoding type")
			}

			// DEBUG Display data
			// data, _ := ioutil.ReadAll(r)
			// log.Println("Data received ", data)

			if err := dec.Decode(q); err == io.EOF {
				log.Println("[DEBUG:MW] End of decoding")
			} else if err != nil {
				log.Println("[ERROR:MW] Request: error while decoding ", err)
				cancel()
			} else {
				log.Println("[DEBUG:MW] Done with success ", q)
			}

			if user, ok := q.(*UserQuery); ok {
				encoder.Encode(user.User)
			}
			if options, ok := q.(*OptionsQuery); ok {
				encoder.Encode(options)
			}
			if proxy, ok := q.(*ProxyQuery); ok {
				encoder.Encode(proxy)
			}
			if token, ok := q.(*TokenQuery); ok {
				encoder.Encode(token)
			}
			if client, ok := q.(*ClientQuery); ok {
				encoder.Encode(client)
			}

			return nil
		},
	}

	return pydioworker.Job(job), nil
}

type decoder interface {
	Decode(v interface{}) error
}
