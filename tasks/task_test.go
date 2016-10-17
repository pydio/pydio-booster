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
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	pydio "github.com/pydio/pydio-booster/io"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	fakeData string
	fakeTask *Task

	secret string
)

func init() {
	secret = "TestingSecret"
	fakeData = `
		{
			"id": "test",
			"flags": 5,
			"label": "Testing",
			"userId": "test",
			"wsId": "test",
			"status": 1,
			"statusMessage": "Testing...",
			"progress": -1,
			"schedule": {"type":null,"value":null},
			"action":"test",
			"parameters": {"param1":"value1","param2":"value2"},
			"nodes":[]
		}`

	fakeTask = &Task{
		ID:            "test",
		Flags:         5,
		Label:         "Testing",
		User:          pydio.User{ID: "test"},
		Repo:          pydio.Repo{ID: "test"},
		Action:        "test",
		Status:        1,
		StatusMessage: "Testing...",
		Progress:      -1,
		Schedule:      Schedule{},
		Parameters: map[string]string{
			"param1": "value1",
			"param2": "value2",
		},
	}
}

func TestSuccess(t *testing.T) {

	Convey("Decoding a task", t, func() {
		dec := json.NewDecoder(strings.NewReader(fakeData))

		var task Task

		// decode an array value (Task)
		err := dec.Decode(&task)
		So(err, ShouldBeNil)

		So(task, ShouldResemble, *fakeTask)
	})

	Convey("Decrypting token", t, func() {

		var b bytes.Buffer

		buf := bufio.NewWriter(&b)

		enc := json.NewEncoder(buf)

		err := enc.Encode(fakeTask)
		So(err, ShouldBeNil)

		buf.Flush()

		var task1, task2 Task

		dec := json.NewDecoder(strings.NewReader(b.String()))
		err = dec.Decode(&task1)
		So(err, ShouldBeNil)

		dec = json.NewDecoder(strings.NewReader(fakeData))
		err = dec.Decode(&task2)
		So(err, ShouldBeNil)

		So(task1, ShouldResemble, task2)
	})
}
