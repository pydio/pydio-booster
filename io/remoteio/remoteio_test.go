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
	"bytes"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/pydio/pydio-booster/encoding/path"
	pydhttp "github.com/pydio/pydio-booster/http"
	pydio "github.com/pydio/pydio-booster/io"
	. "github.com/smartystreets/goconvey/convey"
)

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

func TestAPI(t *testing.T) {

	var node pydio.Node
	path.Unmarshal([]byte("/pydio/testing"), &node)

	Convey("Move a local file to a local location", t, func() {

		apiURL, _ := url.Parse("http://localhost:8080/?get_action=keystore_generate_auth_token")
		api, err := pydhttp.NewAPI(*apiURL, "36g82aa8jplgd8rdevu700tha3")

		So(err, ShouldBeNil)

		writeHandler := Write(api)
		resp, _ := writeHandler(bytes.NewReader([]byte("This is a test")), node)

		So(resp.StatusCode, ShouldEqual, http.StatusOK)

	})
}
