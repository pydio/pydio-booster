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
package s3io

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/pydio/pydio-booster/encoding/path"
	"github.com/pydio/pydio-booster/io"
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

	var node *pydio.Node
	path.Unmarshal([]byte("/s3/tmp/test"), &node)

	node.Options = pydio.Options{
		S3Options: pydio.S3Options{
			APIKey:    "AKIAIJL4RED35JBS2YZA",
			SecretKey: "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",
			Container: "s3pydiotest",
			Region:    "us-east-1",
		},
	}

	// Get at least 10MB or reach end-of-file
	f, err := os.Create("/tmp/test")
	if err != nil {
		log.Fatal(err)
	}

	if err := f.Truncate(1e7); err != nil {
		log.Fatal(err)
	}

	/*fmt.Println("Write to a s3 node")
	Convey("Write to a local node", t, func() {

		file := Open(node, os.O_WRONLY)

		data, _ := ioutil.ReadAll(f)
		bytesWritten, err := file.Write(data)
		So(err, ShouldBeNil)

		file.Close()

		So(bytesWritten, ShouldEqual, 1e7)

	})

	fmt.Println("Append to a s3 node")
	Convey("Append to a local node", t, func() {

		file := Open(node, os.O_WRONLY|os.O_APPEND)

		bytesWritten, err := file.Write([]byte(" Appending content to a file"))
		So(err, ShouldBeNil)

		file.Close()

		So(bytesWritten, ShouldEqual, 28)
	})*/

	fmt.Println("Using WriteAt s3 node")
	Convey("Append to a local node", t, func() {

		file, err := Open(node, os.O_WRONLY)
		So(err, ShouldBeNil)

		var wg sync.WaitGroup
		for r := 0; r < 10; r++ {
			wg.Add(1)
			go func(r int) {
				file.WriteAt([]byte(strconv.Itoa(r)), int64(r))
				wg.Done()
			}(r)
		}
		//So(err, ShouldBeNil)
		wg.Wait()
		file.Close()

		//So(bytesWritten, ShouldEqual, 28)
	})

	fmt.Println("Read from a s3 node")
	Convey("Read from a local node", t, func() {

		file, err := Open(node, os.O_RDONLY)
		So(err, ShouldBeNil)

		bytes, err := ioutil.ReadAll(file)
		So(err, ShouldBeNil)

		file.Close()

		So(len(bytes), ShouldEqual, 1e7+28)

		// Removing file at the end
		os.Remove(file.String())
	})

	// Removing file at the end
	os.Remove(f.Name())
}
