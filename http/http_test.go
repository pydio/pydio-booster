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
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"

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

	u, _ := url.Parse("REPLACE_WITH_PATH") // eg : http://localhost:8080/index.php?get_action=keystore_generate_auth_token
	api, err := NewAPI(*u, "REPLACE_WITH_COOKIE")

	if err != nil {
		fmt.Printf("Received error %v\n", err)
	}

	Convey("Sending a simple request", t, func() {
		client := NewClient()

		uri := "/api/pydio/ws_authenticate"
		auth := api.GetQueryArgs(uri)

		fmt.Printf("Received auth %v\n", api)

		req, err := http.NewRequest("GET", "REPLACE_WITH_PATH"+uri+"?auth_token="+auth.Token+"&auth_hash="+auth.Hash, nil)
		if err != nil {
			t.Fatal(err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}

		code := resp.StatusCode

		So(code, ShouldEqual, http.StatusOK)
		So(err, ShouldBeNil)
	})
}
