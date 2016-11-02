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
	"path"

	pydiopath "github.com/pydio/pydio-booster/encoding/path"
	"github.com/pydio/pydio-booster/encoding/query"
)

// Dir format definition
type Dir struct {
	ParentDir *Dir   `path:",first:last-1"`
	DirName   string `path:",last-1:last" query:"dir,"`
}

// NewDir based on array of strings
func NewDir(dirs ...string) *Dir {

	var str string

	new := Dir{}

	for _, dir := range dirs {
		str = path.Join(str, dir)
	}

	b := []byte(str)

	if err := pydiopath.Unmarshal(b, &new); err == nil {
		return &new
	}

	return nil
}

// Read the node by encoding to its json representation
func (d *Dir) Read(p []byte) (int, error) {

	data, err := json.Marshal(d)

	numBytes := copy(p, data)

	return numBytes, err
}

type dir Dir

// UnmarshalPath implementation
func (d *Dir) UnmarshalPath(b []byte) (err error) {
	new := dir{}

	if err = pydiopath.Unmarshal(b, &new); err == nil {
		if new.ParentDir != nil && new.ParentDir.DirName == "" {
			new.ParentDir = nil
		}
		if new.ParentDir != nil && new.DirName == "" {
			*d = *new.ParentDir
		} else {
			*d = Dir(new)
		}
	}

	return
}

// UnmarshalQuery implementation
func (d *Dir) UnmarshalQuery(b []byte) (err error) {

	new := dir{}

	if err = query.Unmarshal(b, &new); err == nil {
		*d = *NewDir(new.DirName)
		return
	}

	*d = *NewDir(string(b))
	err = nil

	return
}

// String value of the structure Dir
func (d *Dir) String() string {
	if d == nil {
		return ""
	}

	if d.ParentDir != nil {
		return fmt.Sprintf("%s/%s", d.ParentDir, d.DirName)
	}

	return fmt.Sprintf("/%s", d.DirName)
}
