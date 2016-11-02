// Package localio contains logic for dealing with local files
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
package localio

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"testing"

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

var (
	node *pydio.Node
)

func init() {
	node = pydio.NewNode("/my-files/tmp/test")
}

func TestAPI(t *testing.T) {

	Convey("Write to a local node", t, func() {

		file := Open(node, os.O_CREATE|os.O_WRONLY|os.O_EXCL)

		bytesWritten, _ := file.Write([]byte("This is a test"))
		file.Close()

		So(bytesWritten, ShouldEqual, 14)
		compareContents("/tmp/test", []byte("This is a test"))

	})

	Convey("Append to a local node", t, func() {

		file := Open(node, os.O_APPEND|os.O_WRONLY|os.O_EXCL)

		bytesWritten, _ := file.Write([]byte(" Appending content to a file"))
		file.Close()

		So(bytesWritten, ShouldEqual, 28)
		compareContents("/tmp/test", []byte("This is a test Appending content to a file"))

	})

	Convey("Read from a local node", t, func() {

		file := Open(node, os.O_RDONLY|os.O_EXCL)

		bytes, err := ioutil.ReadAll(file)
		So(err, ShouldBeNil)

		file.Close()

		So(bytes, ShouldResemble, []byte("This is a test Appending content to a file"))

		// Removing file at the end
		os.Remove(file.String())
	})

	Convey("Write to a local copy node", t, func() {

		file := Open(node, os.O_CREATE|os.O_WRONLY|os.O_EXCL)

		bytesWritten, _ := io.Copy(file, bytes.NewReader([]byte("This is a test")))
		file.Close()

		So(bytesWritten, ShouldEqual, 14)
		compareContents("/tmp/test", []byte("This is a test"))

		os.Remove(file.String())

	})

}
