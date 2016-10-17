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
package pydio

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	fakeXML  []byte
	fakeXML2 []byte
)

func init() {
	secret = "TestingSecret"

	fakeXML = []byte(`
	<?xml version="1.0" encoding="UTF-8"?>
	<tree>
		<user groupPath="test" id="test">
			<active_repo id="1" write="1" read="1"/>
			<repositories>
				<repo id="test"></repo>
			</repositories>
		</user>
	</tree>
	`)
}

type Query struct {
	User User `xml:"user"`
}

func TestUser(t *testing.T) {
	Convey("Unmarshaling user", t, func() {
		var q Query

		err := xml.Unmarshal(fakeXML, &q)
		if err != nil {
			fmt.Printf("error: %v", err)
			return
		}

		So(q.User, ShouldResemble, *fakeUser)
	})

	Convey("Unmarshaling user 2", t, func() {
		var q Query

		dec := xml.NewDecoder(bytes.NewReader(fakeXML))
		dec.Strict = false

		err := dec.Decode(&q)
		So(err, ShouldBeNil)

		So(q.User, ShouldResemble, *fakeUser)
	})
}
