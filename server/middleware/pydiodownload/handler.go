// Package pydioupload contains the logic for the pydioupload caddy directive
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
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
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

const (
	redirectHeader        string = "X-Accel-Redirect"
	contentLengthHeader   string = "Content-Length"
	contentEncodingHeader string = "Content-Encoding"
	maxRedirectCount      int    = 10
)

func isInternalRedirect(w http.ResponseWriter) bool {
	return w.Header().Get(redirectHeader) != ""
}

// ServerHTTP Requests for downloading files from the server
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {

	switch r.Method {
	case http.MethodGet:
		for _, rule := range h.Rules {
			if httpserver.Path(r.URL.Path).Matches(rule.Path) {

				res := errHandle(r, handle(w, r, h.Dispatcher))

				if res.Err != nil {
					logger.Errorln("returns error : ", res.Err)
					return http.StatusUnauthorized, res.Err
				}

				r = r.WithContext(res.Context)

				return http.StatusOK, nil
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

func handle(w http.ResponseWriter, r *http.Request, d *pydioworker.Dispatcher) func() *pydhttp.Status {

	return func() *pydhttp.Status {

		logger.Infoln("REQ START")

		start := time.Now()

		defer func() {
			elapsed := time.Since(start)
			logger.Infof("REQ END took %s", elapsed)
		}()

		ctx := r.Context()

		// Retrieving the node
		var node *pydio.Node
		if err := getValue(ctx, "node", &node); err != nil {
			return pydhttp.NewStatusErr(http.StatusInternalServerError, err)
		}
		logger.Debugf("Context node : %v", node)

		// Retrieving the options
		var options *pydio.Options
		if err := getValue(ctx, "options", &options); err != nil {
			return pydhttp.NewStatusErr(http.StatusInternalServerError, err)
		}
		logger.Debugf("Context Options : %v", options)

		if options.Path == "" {
			return pydhttp.NewStatusErr(http.StatusFailedDependency, errors.New("Could not retrieve the context node or context options"))
		}

		dir, name := path.Split(options.Path)

		node = pydio.NewNode(
			node.Repo.String(),
			dir,
			name,
		)

		node.Options = *options

		// Refreshing context
		ctx = pydhttp.NewContext(ctx, "node", node)

		// Local file system, creating the Node
		var file *pydio.File
		var err error
		if options.FileOptions.Type == "fs" || options.FileOptions.Type == "local" {
			localNode := pydio.NewNode("local", options.FileOptions.Path, dir, name)
			file, err = localio.Open(localNode, os.O_RDONLY)
		} else if options.FileOptions.Type == "s3" {
			file, err = s3io.Open(node, os.O_RDONLY)
		}

		if err != nil {
			return pydhttp.NewStatusErr(http.StatusUnauthorized, err)
		}

		defer func() {
			if file != nil {
				file.Close()
			}
		}()

		// NEED TO COPY TO BUFFER
		w.Header().Set("Content-Type", "application/octet-stream");
		w.Header().Set("Content-Disposition", "attachment; filename=" + name);

		io.Copy(w, file)

		return pydhttp.NewStatusOK(r, ctx)
	}
}

// asynchronously retrieve values sitting in the context
func getValue(ctx context.Context, key string, value interface{}) error {

	// var node *pydio.Node
	var buf bytes.Buffer

	if err := pydhttp.FromContext(ctx, key, &buf); err != nil {
		return err
	}

	data := buf.String()
	if unquoted, err := strconv.Unquote(strings.Trim(data, "\n")); err == nil {
		data = unquoted
	}

	dec := json.NewDecoder(strings.NewReader(data))
	if err := dec.Decode(&value); err != nil {
		logger.Errorf("value for %s : %v", key, err)
		return err
	}

	return nil
}

// internalResponseWriter wraps the underlying http.ResponseWriter and ignores
// calls to Write and WriteHeader if the response should be redirected to an
// internal location.
type internalResponseWriter struct {
	http.ResponseWriter
}

// ClearHeader removes script headers that would interfere with follow up
// redirect requests.
func (w internalResponseWriter) ClearHeader() {
	w.Header().Del(redirectHeader)
	w.Header().Del(contentLengthHeader)
	w.Header().Del(contentEncodingHeader)
}

// WriteHeader ignores the call if the response should be redirected to an
// internal location.
func (w internalResponseWriter) WriteHeader(code int) {
	if !isInternalRedirect(w) {
		w.ResponseWriter.WriteHeader(code)
	}
}

// Write ignores the call if the response should be redirected to an internal
// location.
func (w internalResponseWriter) Write(b []byte) (int, error) {
	if isInternalRedirect(w) {
		return 0, nil
	}
	return w.ResponseWriter.Write(b)
}

// Rule for the uploader
type (
	Rule struct {
		Path string
	}
)
