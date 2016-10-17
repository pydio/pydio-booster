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
package pydioupload

import (
	"bytes"
	"log"

	"github.com/pydio/pydio-booster/io"
	"github.com/pydio/go/io"
)

// Job definition for the uploader
type Job struct {
	File     *pydio.File
	Buf      bytes.Buffer
	Offset   int64
	NumBytes int64
}

// Do the job
func (j *Job) Do() (err error) {
	defer func() {
		j.File.Done()
	}()

	_, err = j.File.WriteAt(j.Buf.Bytes(), j.Offset)
	if err != nil {
		log.Println("Error here", err)
		return err
	}

	return
}
