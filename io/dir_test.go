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
	"fmt"
	"testing"

	"github.com/pydio/pydio-booster/encoding/path"
	"github.com/pydio/pydio-booster/encoding/query"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	noDir    *Dir
	emptyDir *Dir
	oneDir   *Dir
	threeDir *Dir
	nDir     *Dir
)

func init() {
	noDir = NewDir()
	emptyDir = NewDir("")
	oneDir = NewDir("/dir1")
	threeDir = NewDir("/dir1/dir2/dir3")
	nDir = NewDir("/dir1/dir2/dir3/dir4/dir5/dir6/dir7/dir8/dir9/dir10/dir11/dir12/dir13/dir14/dir15/dir16/dir17/dir18/dir19/dir20")
}

// TestDir function
func TestDir(t *testing.T) {

	fmt.Println("Dir creation")

	Convey("Test creating a Path with a weird format", t, func() {
		dir := NewDir("/////")

		So(dir, ShouldResemble, emptyDir)
		So(dir, ShouldResemble, noDir)
	})
}

// TestPathUnmarshalDir
func TestPathUnmarshalDir(t *testing.T) {

	fmt.Println("Path Unmarshalling")

	Convey("Testing unmarshalling of a Path with no dir", t, func() {
		var urlBlob = []byte("")
		var dir Dir
		err := path.Unmarshal(urlBlob, &dir)
		So(err, ShouldBeNil)

		So(dir, ShouldResemble, *noDir)
	})

	Convey("Testing unmarshalling of a Path with an empty dir", t, func() {
		var urlBlob = []byte("/")
		var dir Dir
		err := path.Unmarshal(urlBlob, &dir)
		So(err, ShouldBeNil)

		So(dir, ShouldResemble, *emptyDir)
	})

	Convey("Testing unmarshalling of a Path with a one-level dir", t, func() {
		var urlBlob = []byte("/dir1")
		var dir Dir
		err := path.Unmarshal(urlBlob, &dir)
		So(err, ShouldBeNil)

		So(dir, ShouldResemble, *oneDir)
	})

	Convey("Testing unmarshalling of a Path with an n-level dir", t, func() {
		var urlBlob = []byte("/dir1/dir2/dir3/dir4/dir5/dir6/dir7/dir8/dir9/dir10/dir11/dir12/dir13/dir14/dir15/dir16/dir17/dir18/dir19/dir20")

		var dir Dir
		err := path.Unmarshal(urlBlob, &dir)
		So(err, ShouldBeNil)

		So(dir, ShouldResemble, *nDir)
	})

	Convey("Testing unmarshalling of a Path with an n-level dir and a bizarrely formatted path", t, func() {
		var urlBlob = []byte("/////dir1/dir2//dir3/dir4////////dir5/dir6//dir7/dir8/dir9/dir10/dir11/dir12/dir13/dir14/dir15/dir16/dir17/dir18/dir19/dir20///")

		var dir Dir
		err := path.Unmarshal(urlBlob, &dir)
		So(err, ShouldBeNil)

		So(dir, ShouldResemble, *nDir)
	})
}

// TestQueryUnmarshalDir
func TestQueryUnmarshalDir(t *testing.T) {

	fmt.Println("Query Unmarshalling")

	Convey("Testing unmarshalling of a Query with no dir", t, func() {
		var urlBlob = []byte("")
		var dir Dir
		err := query.Unmarshal(urlBlob, &dir)
		So(err, ShouldBeNil)

		So(dir, ShouldResemble, *noDir)
	})

	Convey("Testing unmarshalling of a Query with a one-level dir", t, func() {
		var urlBlob = []byte("?dir=/dir1")
		var dir Dir
		err := query.Unmarshal(urlBlob, &dir)
		So(err, ShouldBeNil)

		So(dir, ShouldResemble, *oneDir)
	})

	Convey("Testing unmarshalling of a Query with a 3-level dir", t, func() {
		var urlBlob = []byte(`?dir=/dir1/dir2/dir3`)
		var dir Dir
		err := query.Unmarshal(urlBlob, &dir)
		So(err, ShouldBeNil)

		So(dir, ShouldResemble, *threeDir)
	})

	Convey("Testing unmarshalling of a Query with an n-level dir", t, func() {
		var urlBlob = []byte("?dir=/dir1/dir2/dir3/dir4/dir5/dir6/dir7/dir8/dir9/dir10/dir11/dir12/dir13/dir14/dir15/dir16/dir17/dir18/dir19/dir20")

		var dir Dir
		err := query.Unmarshal(urlBlob, &dir)
		So(err, ShouldBeNil)

		So(dir, ShouldResemble, *nDir)
	})

	Convey("Testing unmarshalling of a Query with an n-level dir and a bizarrely formatted path", t, func() {
		var urlBlob = []byte("?dir=/////dir1/dir2//dir3/dir4////////dir5/dir6//dir7/dir8/dir9/dir10/dir11/dir12/dir13/dir14/dir15/dir16/dir17/dir18/dir19/dir20///")

		var dir Dir
		err := query.Unmarshal(urlBlob, &dir)
		So(err, ShouldBeNil)

		So(dir, ShouldResemble, *nDir)
	})
}
