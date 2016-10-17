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
	"io"
	"log"
	"sync"
)

// File format definition
type File struct {
	Node *Node
	str  string

	*Reader
	Writer
	lock chan (int)

	sync.WaitGroup
}

// Reader pydio style
type Reader struct {
	//io.ReaderAt
	io.ReadSeeker

	*io.PipeReader
}

// Writer pydio style
type Writer interface {
	io.Writer
	io.Closer
	io.WriterAt
}

// NewFile from a node
func NewFile(node *Node, str string, reader *Reader, w interface{}, writeLock chan (int)) *File {

	var writer Writer
	var ok bool

	if writer, ok = w.(Writer); !ok {
		writer = nil
	}

	return &File{
		Node:   node,
		str:    str,
		Reader: reader,
		Writer: writer,
		lock:   writeLock,
	}
}

func (f *File) String() string {
	return f.str
}

// Close pipe
func (f *File) Close() {
	if f.Writer != nil {

		// Waiting for all potential tasks to be finished
		f.Wait()

		defer func() {
			// Creating a lock to wait for the pipe reader to close
			if f.lock != nil {
				<-f.lock
			}
		}()

		f.Writer.Close()
	} else {
		log.Println("Empty")
	}
}

//
func (r *Reader) Read(p []byte) (n int, err error) {
	return r.PipeReader.Read(p)
}

// Seek faking
func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}
