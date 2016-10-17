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
package localio

import (
	"io"
	"log"
	"os"
	"path"

	pydio "github.com/pydio/pydio-booster/io"
)

// Open local file node
func Open(node *pydio.Node, flag int) (*pydio.File, error) {

	var reader *pydio.Reader
	var writer io.Writer

	// For a local file, the name of the repo is dropped
	name := path.Join(node.Dir.String(), node.Basename)

	// Opening the file
	file, err := os.OpenFile(name, flag, 0666)
	if err != nil {
		log.Println("Could not open file", err)
		return nil, err
	}

	log.Println("Opened file ", name)

	// Creating the handlers
	if flag&os.O_RDWR != 0 || flag&os.O_WRONLY == 0 {
		reader = readHandler(file)
	}

	if flag&os.O_WRONLY != 0 || flag&os.O_RDWR != 0 {
		log.Println("We have a write handler")
		writer = writeHandler(file)
	}

	return pydio.NewFile(
		node,
		name,
		reader,
		writer,
		nil,
	), nil
}

func readHandler(file *os.File) *pydio.Reader {
	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()

		io.Copy(writer, file)
	}()

	return &pydio.Reader{PipeReader: reader}
}

func writeHandler(file *os.File) io.Writer {
	return file
}
