// Package pydio contains all objects needed by the Pydio system
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
	"encoding/json"
	"fmt"
	"os"
	"path"

	pydiopath "github.com/pydio/pydio-booster/encoding/path"
	"github.com/pydio/pydio-booster/encoding/query"
)

// Node format definition
type Node struct {
	Repo     *Repo  `path:",0:1" query:"repo"`
	Dir      *Dir   `path:",1:last-1" query:"dir"`
	Basename string `path:",last-1:last" query:"file"`

	Options Options `json:"-"`
}

// NewNode from a bunch of string (or concatenated ones)
func NewNode(items ...string) *Node {
	var str string

	new := Node{}

	for _, item := range items {
		str = path.Join(str, item)
	}

	b := []byte(str)

	if err := pydiopath.Unmarshal(b, &new); err == nil {
		return &new
	}

	return nil
}

// NewTmpNode with random name
func NewTmpNode() (*Node, error) {
	// Creating a Unique ID for the connection
	u4 := "tmp"
	//uuid.NewV4()
	//if err != nil {
	//	return nil, err
	//}

	dir := NewDir(os.TempDir())

	return &Node{
		Dir:      dir,
		Basename: u4, //.String(),
	}, nil
}

// String representation of a node
func (n *Node) String() string {

	if n.Dir != nil {
		return fmt.Sprintf("pydio://%s/%s/%s", n.Repo, n.Dir, n.Basename)
	}

	return fmt.Sprintf("pydio://%s/%s", n.Repo, n.Basename)
}

// Read the node by encoding to its json representation
func (n *Node) Read(p []byte) (int, error) {

	data, err := json.Marshal(n)

	numBytes := copy(p, data)

	return numBytes, err
}

type node Node

// UnmarshalPath implementation
func (n *Node) UnmarshalPath(b []byte) (err error) {
	new := node{}

	if err = pydiopath.Unmarshal(b, &new); err == nil {
		*n = Node(new)
		return
	}

	//*n = Node{string(b)}
	return
}

// UnmarshalQuery implementation
func (n *Node) UnmarshalQuery(b []byte) (err error) {
	new := node{}

	if err = query.Unmarshal(b, &new); err == nil {
		*n = Node(new)
		return
	}

	//*n = Node{string(b)}
	return
}
