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
package main

import (
	"bytes"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	scratchDir    string // tests will create files and directories here
	trivialConfig string
)

func init() {
}

// Generates a new temporary file name without a path.
func tempFileName() string {
	buffer := make([]byte, 16)
	_, _ = rand.Read(buffer)
	for i := range buffer {
		buffer[i] = (buffer[i] % 25) + 97 // a–z
	}
	return string(buffer)
}

func compareContents(filename string, contents []byte) {
	fd, err := os.Open(filename)
	So(err, ShouldBeNil)
	if err != nil {
		return
	}
	defer fd.Close()

	buffer := make([]byte, (len(contents)/4096+1)*4096)
	n, err := fd.Read(buffer)
	if err != nil {
		SkipSo(n, ShouldEqual, len(contents))
		SkipSo(buffer[0:len(contents)], ShouldResemble, contents)
		So(err, ShouldBeNil)
		return
	}
	So(n, ShouldEqual, len(contents))
	So(buffer[0:len(contents)], ShouldResemble, contents)
}

func TestUpload(t *testing.T) {
	Convey("Uploading files using POST", t, func() {
		Convey("succeeds with two trivially small files", func() {
			tempFName := tempFileName()

			// START
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			p, _ := writer.CreateFormFile("A", tempFName)
			p.Write([]byte("DELME"))
			writer.Close()
			// END

			req, err := http.NewRequest("POST", "http://localhost/io/my-files/dir1", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			// Setting COOKIE
			/*expire := time.Now().AddDate(0, 0, 1)

			sessionCookie := &http.Cookie{
				Name:    "AjaXplorer",
				Expires: expire,
				Value:   "98ep0nie19e25qis01g8cunl47",
			}

			req.AddCookie(sessionCookie)*/
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				os.Remove(filepath.Join(scratchDir, tempFName))
			}()

			client := http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			So(resp.StatusCode, ShouldEqual, 200)
		})
	})
}
