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
package query

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type Node struct {
	R string `query:"repo,"`
	D string `query:"dir,"`
	F string `query:"file,"`
}

var (
	fakeNode *Node
)

func init() {
	fakeNode = &Node{
		R: "my-files",
		D: "/dir1/dir2/dir3",
		F: "file1.txt",
	}

}

/*
func ExampleMarshal() {
	type ColorGroup struct {
		ID     int
		Name   string
		Colors []string
	}
	group := ColorGroup{
		ID:     1,
		Name:   "Reds",
		Colors: []string{"Crimson", "Red", "Ruby", "Maroon"},
	}
	b, err := json.Marshal(group)
	if err != nil {
		fmt.Println("error:", err)
	}
	os.Stdout.Write(b)
	// Output:
	// {"ID":1,"Name":"Reds","Colors":["Crimson","Red","Ruby","Maroon"]}
}*/

func TestUnmarshalQueryParams2(t *testing.T) {

	var urlBlob = []byte(`?repo=my-files&dir=/dir1/dir2/dir3&file=file1.txt`)

	Convey("Testing unmarshalling with a standard structure", t, func() {
		var node Node
		err := Unmarshal(urlBlob, &node)
		So(err, ShouldBeNil)

		So(node, ShouldResemble, *fakeNode)
	})
}
