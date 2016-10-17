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
package path

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type Dir []string

type ComplexNode struct {
	R string `path:"repo,0:1"`
	D Dir    `path:"dir,1:last-1"`
	F string `path:"file,last-1:last"`
}

type Node struct {
	R string   `path:"repo,0:1"`
	D []string `path:"dir,1:last-1"`
	F string   `path:"file,last-1:last"`
}

type SimpleNode struct {
	S string `path:"str,first:last"`
}

type Query struct {
	Action string `path:"action,0:1"`
	Node   Node   `path:"node,1:last"`
}

type node Node

func (n *Node) UnmarshalPath(b []byte) (err error) {
	new := node{}

	if err = Unmarshal(b, &new); err == nil {
		*n = Node(new)
		return
	}

	return
}

var (
	fakeNode        *Node
	fakeSimpleNode  *SimpleNode
	fakeComplexNode *ComplexNode
)

func init() {
	fakeComplexNode = &ComplexNode{
		R: "my-files",
		D: Dir{
			"dir1", "dir2", "dir3",
		},
		F: "file1.txt",
	}

	fakeNode = &Node{
		R: "my-files",
		D: []string{"dir1", "dir2", "dir3"},
		F: "file1.txt",
	}

	fakeSimpleNode = &SimpleNode{
		fakeNode.R + "/" + strings.Join(fakeNode.D, "/") + "/" + fakeNode.F,
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

func TestUnmarshal(t *testing.T) {

	var urlBlob = []byte(`my-files/dir1/dir2/dir3/file1.txt`)
	Convey("Testing unmarshalling with a simple structure", t, func() {
		var simpleNode SimpleNode
		err := Unmarshal(urlBlob, &simpleNode)
		So(err, ShouldBeNil)

		So(simpleNode, ShouldResemble, *fakeSimpleNode)
	})

	urlBlob = []byte(`my-files/dir1/`)
	Convey("Testing unmarshalling with a standard structure", t, func() {
		var node Node
		err := Unmarshal(urlBlob, &node)
		So(err, ShouldBeNil)

		So(node.F, ShouldBeEmpty)
	})

	urlBlob = []byte(`/my-files/dir1/dir2/dir3/file1.txt`)
	Convey("Testing unmarshalling with a simple structure", t, func() {
		var simpleNode SimpleNode
		err := Unmarshal(urlBlob, &simpleNode)
		So(err, ShouldBeNil)

		So(simpleNode, ShouldResemble, *fakeSimpleNode)
	})

	Convey("Testing unmarshalling with a standard structure", t, func() {
		var node Node
		err := Unmarshal(urlBlob, &node)
		So(err, ShouldBeNil)

		So(node, ShouldResemble, *fakeNode)
	})

	Convey("Testing unmarshalling with a complex structure", t, func() {
		var node ComplexNode
		err := Unmarshal(urlBlob, &node)
		So(err, ShouldBeNil)

		So(node, ShouldResemble, *fakeComplexNode)
	})

	Convey("Testing nil values", t, func() {
		var urlBlob = []byte(`/my-files/dir1/`)
		var node Node

		err := Unmarshal(urlBlob, &node)
		So(err, ShouldBeNil)
		So(node.F, ShouldBeEmpty)
	})

	Convey("Testing nil values", t, func() {
		var urlBlob = []byte(`/my-files/dir1/file1.txt`)
		var node Node

		err := Unmarshal(urlBlob, &node)
		So(err, ShouldBeNil)
	})

	Convey("Testing multiple unmarshal", t, func() {
		var urlBlob = []byte(`/action/my-files/dir1/`)
		var q Query

		err := Unmarshal(urlBlob, &q)
		So(err, ShouldBeNil)
	})
}
