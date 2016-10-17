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
package pydiotasks

import (
	"encoding/json"

	"github.com/pydio/pydio-booster/io"
	"github.com/pydio/go/io"
)

// Task structure
type Task struct {
	ID            string            `json:"id"`
	Flags         int               `json:"flags"`
	Label         string            `json:"label"`
	Description   string            `json:"description,omitempty"`
	User          pydio.User        `json:"userId"`
	Repo          pydio.Repo        `json:"wsId"`
	Status        int               `json:"status"`
	StatusMessage string            `json:"statusMessage"`
	Progress      int               `json:"progress"`
	Schedule      Schedule          `json:"schedule"`
	Action        string            `json:"action"`
	Parameters    map[string]string `json:"parameters"`
}

// Schedule of a task
type Schedule struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// New task
func New(l string, d string, u pydio.User, r pydio.Repo, a string, s Schedule) *Task {
	t := &Task{
		Label:       l,
		Description: d,
		User:        u,
		Repo:        r,
		Action:      a,
		Schedule:    s,
	}

	return t
}

type task Task

// UnmarshalJSON structure into Task
func (t *Task) UnmarshalJSON(b []byte) (err error) {

	var new = task{}

	if err = json.Unmarshal(b, &new); err == nil {
		*t = Task(new)
	}

	return
}
