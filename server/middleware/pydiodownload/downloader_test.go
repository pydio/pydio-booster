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
	"fmt"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/mholt/caddy/caddyhttp/httpserver"
	"github.com/pydio/pydio-booster/io"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	auth     string
	authURL  *url.URL
	tokenURL *url.URL
	config   string
)

func RandomString(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}

	return string(result)
}

func init() {
	auth = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE0NzE3MDUzODUsIm9wdGlvbnMiOiJleUpVV1ZCRklqb2labk1pTENKUVFWUklJam9pTDFWelpYSnpMMmRvWldOeGRXVjBMMU5wZEdWekwyNWhiV1Z6Y0dGalpYTXZZMjl5WlM5emNtTXZaR0YwWVM5d1pYSnpiMjVoYkM5aFpHMXBiaUo5In0.XzUVns-e9SXgNWdqGXCk7f_V6hHoKyt-ymkAmn3X1MM"

	config = `pydioupload /upload`
}

func newTestUploadHandler(t *testing.T, configExcerpt string) httpserver.Handler {
	/*
		c := setup.NewTestController(configExcerpt)
		m, err := Setup(c)
		if err != nil {
			t.Fatal(err)
		}

		next := middleware.HandlerFunc(func(w http.ResponseWriter, r *http.Request) (int, error) {
			return http.StatusTeapot, nil
		})
	*/
	return nil // m(next)
}

func TestUpload_ServeHTTP(t *testing.T) {

	Convey("Testing the upload functionality", t, func() {
		h := newTestUploadHandler(t, config)
		w := httptest.NewRecorder()

		Convey("uploading two small files to a local workspace (my-files)", func() {
			tempFName := RandomString(10)
			tempFName2 := RandomString(10)

			// Creating nodes
			node1 := pydio.NewNode("my-files", tempFName)
			node2 := pydio.NewNode("my-files", tempFName2)

			// Creating Multipart POST data
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			p, _ := writer.CreateFormFile("A", node1.Basename)
			p.Write([]byte("DELME"))

			p, _ = writer.CreateFormFile("B", node2.Basename)
			p.Write([]byte("REMOVEME"))
			writer.Close()

			// Creating Request
			uri := fmt.Sprintf("/upload/%s%s", node1.Repo.ID, node1.Dir.String())

			req, err := http.NewRequest("POST", uri, body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.Header.Set("Authorization", auth)

			So(err, ShouldBeNil)

			code, err := h.ServeHTTP(w, req)

			So(err, ShouldBeNil)
			So(code, ShouldEqual, 200)

			//So(getContents(node1), ShouldResemble, []byte("DELME"))
			//So(getContents(node2), ShouldResemble, []byte("REMOVEME"))
		})

		Convey("uploading chunked files to a local workspace (my-files)", func() {

			tempFName := RandomString(10)

			// Creating node
			node := pydio.NewNode("my-files", tempFName)

			uri := fmt.Sprintf("/upload/%s%s", node.Repo.ID, node.Dir.String())

			// Creating First Multipart POST data
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			writer.WriteField("partial_target_bytesize", "12")
			writer.WriteField("partial_upload", "true")
			writer.WriteField("xhr_uploader", "true")
			writer.WriteField("force_post", "true")
			writer.WriteField("urlencoded_filename", node.Basename)
			writer.WriteField("appendto_urlencoded_part", node.Basename)
			p, _ := writer.CreateFormFile("A", node.Basename)
			p.Write([]byte("DELME "))
			writer.Close()

			// Sending first request
			req, err := http.NewRequest("POST", uri, body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.Header.Set("Authorization", auth)

			So(err, ShouldBeNil)

			code, err := h.ServeHTTP(w, req)

			So(err, ShouldBeNil)
			So(code, ShouldEqual, 200)

			// Creating Second Multipart POST data
			body = &bytes.Buffer{}
			writer = multipart.NewWriter(body)

			writer.WriteField("partial_target_bytesize", "12")
			writer.WriteField("partial_upload", "true")
			writer.WriteField("xhr_uploader", "true")
			writer.WriteField("force_post", "true")
			writer.WriteField("urlencoded_filename", node.Basename)
			writer.WriteField("appendto_urlencoded_part", node.Basename)

			p, _ = writer.CreateFormFile("B", node.Basename)
			p.Write([]byte("PLEASE"))
			writer.Close()

			// Sending second request
			req, err = http.NewRequest("POST", uri, body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.Header.Set("Authorization", auth)

			So(err, ShouldBeNil)

			code, err = h.ServeHTTP(w, req)

			So(err, ShouldBeNil)
			So(code, ShouldEqual, 200)

		})

		Convey("uploading to a remote location (S3)", func() {
			tempFName := RandomString(10)

			// Creating node
			node := pydio.NewNode("s3", tempFName)

			// Creating Multipart POST data
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			p, _ := writer.CreateFormFile("A", node.Basename)
			p.Write([]byte("DELME"))
			writer.Close()

			// Creating Request
			uri := fmt.Sprintf("/upload/%s%s", node.Repo.ID, node.Dir.String())

			req, err := http.NewRequest("POST", uri, body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			So(err, ShouldBeNil)

			code, err := h.ServeHTTP(w, req)

			So(err, ShouldBeNil)
			So(code, ShouldEqual, 200)

		})

	})
}
