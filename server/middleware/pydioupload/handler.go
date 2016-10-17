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
package pydioupload

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/mholt/caddy/caddyhttp/httpserver"
	"github.com/pydio/pydio-booster/http"
	"github.com/pydio/pydio-booster/io"
	"github.com/pydio/pydio-booster/io/localio"
	"github.com/pydio/pydio-booster/io/s3io"
	"github.com/pydio/pydio-booster/worker"
)

// Handler structure
type Handler struct {
	Next       httpserver.Handler
	Rules      []Rule
	Dispatcher *pydioworker.Dispatcher
}

// ServerHTTP Requests for uploading files to the server
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {

	log.Println("PydioUpload: ServeHTTP")

	switch r.Method {
	case http.MethodOptions:
		for _, rule := range h.Rules {
			if httpserver.Path(r.URL.Path).Matches(rule.Path) {
				return http.StatusOK, nil
			}
		}
	case http.MethodPost:
		for _, rule := range h.Rules {
			if httpserver.Path(r.URL.Path).Matches(rule.Path) {

				res := errHandle(r, handle(r, h.Dispatcher))

				if res.Err != nil {
					log.Println("Pydio Upload returns an error : ", res.Err)
					return http.StatusUnauthorized, res.Err
				}

				r = r.WithContext(res.Context)
			}
		}
	}

	return h.Next.ServeHTTP(w, r)
}

func errHandle(r *http.Request, f func() *pydhttp.Status) *pydhttp.Status {

	ctx := r.Context()

	c := make(chan *pydhttp.Status, 1)

	if err := ctx.Err(); err != nil {
		return pydhttp.NewStatusErr(http.StatusInternalServerError, err)
	}

	go func() { c <- f() }()

	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != nil {
			return pydhttp.NewStatusErr(http.StatusInternalServerError, err)
		}
	case res := <-c:
		return res
	}

	return pydhttp.NewStatusOK(r)
}

func handle(r *http.Request, d *pydioworker.Dispatcher) func() *pydhttp.Status {

	return func() *pydhttp.Status {

		log.Println("[REQ:UPLOAD] START")

		start := time.Now()

		defer func() {
			elapsed := time.Since(start)
			log.Printf("[REQ:UPLOAD] END took %s", elapsed)
		}()

		ctx := r.Context()

		fileOptionsMap := make(map[string]interface{})

		mr, err := r.MultipartReader()
		if err != nil {
			return pydhttp.NewStatusErr(http.StatusInternalServerError, err)
		}

		for {
			var p *multipart.Part

			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}

			if p == nil {
				break
			}

			// Retrieving all options
			formName := p.FormName()
			fileName := p.FileName()

			if formName != "" && fileName == "" {

				var buf []byte
				var b bool
				var i int64

				p.Read(buf)

				if b, err = strconv.ParseBool(string(buf)); err == nil {
					fileOptionsMap[formName] = b
				} else {
					if i, err = strconv.ParseInt(string(buf), 10, 64); err == nil {
						fileOptionsMap[formName] = i
					} else {
						fileOptionsMap[formName] = string(buf)
					}
				}

				continue
			}

			if fileName != "" {

				// Retrieving the node
				var node = &pydio.Node{}
				if err = pydhttp.FromContext(ctx, "node", node); err != nil {
					return pydhttp.NewStatusErr(http.StatusInternalServerError, err)
				}

				// Retrieving the options
				var options = &pydio.Options{}
				if err = pydhttp.FromContext(ctx, "options", options); err != nil {
					return pydhttp.NewStatusErr(http.StatusInternalServerError, err)
				}
				log.Println("[DEBUG:UPLOAD] Context Options ", options)

				// Retrieving request options
				if options.PartialUpload {
					fileName = fmt.Sprintf("%s.dpart", fileName)
				}

				dir, err := url.QueryUnescape(path.Dir(options.Path))
				name, err := url.QueryUnescape(path.Base(options.Path))
				node = pydio.NewNode(
					node.Repo.String(),
					dir,
					name,
				)

				node.Options = *options

				// Refreshing context
				ctx = pydhttp.NewContext(ctx, "node", node)
				ctx = pydhttp.NewContext(ctx, "options", options)

				// Local file system, creating the Node
				var file *pydio.File
				if options.FileOptions.Type == "fs" {
					localNode := pydio.NewNode("local", options.FileOptions.Path, dir, name)
					log.Println("HERE ", options.FileOptions.Path, options.Path, name, localNode)
					file, err = localio.Open(localNode, os.O_CREATE|os.O_WRONLY)
				} else if options.FileOptions.Type == "s3" {
					file, err = s3io.Open(node, os.O_CREATE|os.O_WRONLY)
				}

				if err != nil {
					return pydhttp.NewStatusErr(http.StatusUnauthorized, err)
				}

				defer func() {
					if file != nil {
						file.Close()
					}
				}()

				offset := int64(0)

				for {
					var b bytes.Buffer
					var n int64

					// 1 MB buffer
					n, err = io.CopyN(&b, p, 1*1024*1024)
					if err != nil && err != io.EOF {
						break
					}

					job := &Job{
						File:     file,
						Buf:      b,
						Offset:   offset,
						NumBytes: n,
					}

					file.Add(1)
					d.Add(job)

					offset += n

					if err == io.EOF {
						break
					}
				}
			}
		}

		log.Println("[DEBUG:UPLOAD] ", ctx.Value("node"))
		return pydhttp.NewStatusOK(r, ctx)
	}
}

// Rule for the uploader
type (
	Rule struct {
		Path string
	}
)
