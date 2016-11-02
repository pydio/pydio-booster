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

	"github.com/pydio/pydio-booster/encoding/path"
)

// Repo format definition
type Repo struct {
	ID string `xml:"id,attr" json:"id" path:",first:last"`
}

type repo Repo

// UnmarshalPath implementation
func (r *Repo) UnmarshalPath(b []byte) (err error) {
	new := repo{}

	if err = path.Unmarshal(b, &new); err == nil {
		*r = Repo(new)
		return
	}

	*r = Repo{string(b)}
	return
}

// UnmarshalQuery implementation
func (r *Repo) UnmarshalQuery(b []byte) (err error) {
	new := repo{}

	if err = path.Unmarshal(b, &new); err == nil {
		*r = Repo(new)
		return
	}

	*r = Repo{string(b)}
	return
}

// UnmarshalJSON structure into a Repo
func (r *Repo) UnmarshalJSON(b []byte) (err error) {
	new, s := repo{}, ""

	if err = json.Unmarshal(b, &new); err == nil {
		*r = Repo(new)
		return
	}

	if err = json.Unmarshal(b, &s); err == nil {
		r.ID = s
	}

	return
}

// MarshalJSON representation of user
func (r *Repo) MarshalJSON() ([]byte, error) {
	if r.ID != "" {
		return json.Marshal(r.ID)
	}

	return json.Marshal(nil)
}

// String representation of a repo
func (r *Repo) String() string {
	if r == nil {
		return ""
	}

	return fmt.Sprintf("%s", r.ID)
}
