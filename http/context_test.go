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
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	pydio "github.com/pydio/pydio-booster/io"
	. "github.com/smartystreets/goconvey/convey"
)

type key string

const (
	firstKey  key = "first"
	secondKey key = "second"
	thirdKey  key = "third"
)

var (
	ctx  context.Context
	str  string
	node *pydio.Node
)

func init() {
	ctx = context.Background()
	str = "/tmp/test/testing"
	node = pydio.NewNode(str)
}

func TestContext(t *testing.T) {
	Convey("Writing a string in the context and reading it back", t, func() {
		var localStr string

		// Creating context value (pipe)
		v := NewContextValue()
		ctx = context.WithValue(ctx, firstKey, v)

		// Go Routine that will write the string (strings Reader)
		go func() {
			io.Copy(v, strings.NewReader(str))
			v.Close()
		}()

		// Reading from the context value and write it to a local node (buffer)
		buf := bytes.NewBufferString(localStr)
		err := FromContext(ctx, firstKey, buf)

		So(err, ShouldBeNil)
		So(buf.String(), ShouldEqual, str)
	})

	Convey("Writing a node in the context and reading it back", t, func() {
		var localJSON string
		var localNode *pydio.Node

		// Creating context value (pipe)
		v := NewContextValue()
		ctx = context.WithValue(ctx, secondKey, v)

		// Go Routine that will write the node (encoding)
		go func() {
			enc := json.NewEncoder(v)
			enc.Encode(node)
			v.Close()
		}()

		// Reading from the context value and write it to a local node (decoding)
		buf := bytes.NewBufferString(localJSON)
		err := FromContext(ctx, secondKey, buf)
		So(err, ShouldBeNil)

		dec := json.NewDecoder(buf)
		err = dec.Decode(&localNode)
		So(err, ShouldBeNil)

		// Tests
		So(localNode, ShouldResemble, node)
	})

	Convey("Writing a string in context and reading it back as a node", t, func() {
		var localJSON string
		var localNode *pydio.Node

		v := NewContextValue()
		ctx = context.WithValue(ctx, thirdKey, v)

		go func() {
			io.Copy(v, strings.NewReader(str))
			v.Close()
		}()

		buf := bytes.NewBufferString(localJSON)
		err := FromContext(ctx, thirdKey, buf)
		So(err, ShouldBeNil)

		localNode = pydio.NewNode(buf.String())

		So(localNode, ShouldResemble, node)
	})
}
