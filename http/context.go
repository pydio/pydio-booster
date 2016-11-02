// Package pydhttp contains all http related work
/* Copyright 2007-2016 Abstrium <contact (at) pydio.com>
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
package pydhttp

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"

	"golang.org/x/net/context"
)

// ContextValue Pipe and buffer
type ContextValue struct {
	reader io.Reader
	writer io.Writer

	closed bool

	buf []byte
	off int64
}

// NewContext with the key value
func NewContext(ctx context.Context, key string, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}

// FromContext value of the given key
func FromContext(ctx context.Context, key string, value interface{}) (err error) {

	if reader, ok := ctx.Value(key).(io.Reader); ok {
		dec := json.NewDecoder(reader)
		err = dec.Decode(value)
	} else {
		err = errors.New("Cannot convert this thing to io.Reader")
	}

	return
}

// NewContextValue object
func NewContextValue() *ContextValue {

	reader, writer := io.Pipe()

	return &ContextValue{
		reader: reader,
		writer: writer,
	}
}

// Read a context value, first from the Pipe if it hasn't been read,
// then from the buffer
func (c *ContextValue) Read(p []byte) (n int, err error) {

	if !c.closed {
		var data []byte

		data, _ = ioutil.ReadAll(c.reader)

		n = len(data)

		c.buf = make([]byte, n)
		n = copy(c.buf, data)

		go c.Close()
	}

	n = copy(p, c.buf[c.off:])
	c.off += int64(n)

	if n < len(p) {
		err = io.EOF
		c.off = 0
	}

	return
}

// Close the pipe writer
func (c *ContextValue) Close() error {
	c.closed = true

	if pipeWriter, ok := c.writer.(*io.PipeWriter); ok {
		return pipeWriter.Close()
	}

	return nil
}

// Write content to the pipe end
func (c *ContextValue) Write(p []byte) (n int, err error) {
	return c.writer.Write(p)
}
