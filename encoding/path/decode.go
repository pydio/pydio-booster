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
	"encoding"
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

// Unmarshal parses the Path-encoded data and stores the result
// in the value pointed to by v.
//
func Unmarshal(data []byte, v interface{}) error {
	// Check for well-formedness.
	// Avoids filling out half a data structure
	// before discovering a JSON syntax error.
	var d decodeState
	err := checkValid(data, &d.scan)
	if err != nil {
		return err
	}

	d.init(data)
	return d.unmarshal(v)
}

// Unmarshaler is the interface implemented by objects
// that can unmarshal a Path description of themselves.
type Unmarshaler interface {
	UnmarshalPath([]byte) error
}

// An UnmarshalTypeError describes a Path value that was
// not appropriate for a value of a specific Go type.
type UnmarshalTypeError struct {
	Value  string       // description of Path value - "bool", "array", "number -5"
	Type   reflect.Type // type of Go value it could not be assigned to
	Offset int64        // error occurred after reading Offset bytes
}

func (e *UnmarshalTypeError) Error() string {
	return "url: cannot unmarshal " + e.Value + " into Go value of type " + e.Type.String()
}

// An InvalidUnmarshalError describes an invalid argument passed to Unmarshal.
// (The argument to Unmarshal must be a non-nil pointer.)
type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "url: Unmarshal(nil)"
	}

	if e.Type.Kind() != reflect.Ptr {
		return "url: Unmarshal(non-pointer " + e.Type.String() + ")"
	}
	return "url: Unmarshal(nil " + e.Type.String() + ")"
}

func (d *decodeState) unmarshal(v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &InvalidUnmarshalError{reflect.TypeOf(v)}
	}

	d.scan.reset()

	// We decode rv not rv.Elem because the Unmarshaler interface
	// test must be applied at the top level of the value.
	d.value(rv) // Path

	return d.savedError
}

// decodeState represents the state while decoding a JSON value.
type decodeState struct {
	data       []byte
	off        int // read offset in data
	scan       scanner
	nextscan   scanner // for calls to nextValue
	savedError error
	useNumber  bool
}

// errPhase is used for errors that should not happen unless
// there is a bug in the JSON decoder or something is editing
// the data slice while the decoder executes.
var errPhase = errors.New("Path decoder out of sync - data changing underfoot?")

func (d *decodeState) init(data []byte) *decodeState {
	d.data = data
	d.off = 0
	d.savedError = nil
	return d
}

// error aborts the decoding by panicking with err.
func (d *decodeState) error(err error) {
	panic(err)
}

// saveError saves the first err it is called with,
// for reporting at the end of the unmarshal.
func (d *decodeState) saveError(err error) {
	if d.savedError == nil {
		d.savedError = err
	}
}

// next cuts off and returns the next full Path value in d.data[d.off:].
// The next value is known to be an object or array, not a literal.
func (d *decodeState) next() []byte {
	item, rest, err := nextValue(d.data[d.off:], &d.nextscan)
	if err != nil {
		d.error(err)
	}
	d.off = len(d.data) - len(rest)

	return item
}

// scanWhile processes bytes in d.data[d.off:] until it
// receives a scan code not equal to op.
// It updates d.off and returns the new scan code.
func (d *decodeState) scanWhile(op int) int {
	var newOp int
	for {
		if d.off >= len(d.data) {
			newOp = d.scan.eof()
			d.off = len(d.data) + 1 // mark processed EOF with len+1
		} else {
			c := d.data[d.off]
			d.off++
			newOp = d.scan.step(&d.scan, c)
		}
		if newOp != op {
			break
		}
	}
	return newOp
}

// value decodes a Path value from d.data[d.off:] into the value.
// it updates d.off to point past the decoded value.
func (d *decodeState) value(v reflect.Value) {

	// If the reflect Value doesn't exist, skip to next value
	if !v.IsValid() {
		_, rest, err := nextValue(d.data[d.off:], &d.nextscan)

		if err != nil {
			d.error(err)
		}
		d.off = len(d.data) - len(rest)

		// d.scan thinks we're still at the beginning of the item.
		// Feed in an empty string - the shortest, simplest value -
		// so that it knows we got to the end of the value.
		if d.scan.redo {
			// rewind.
			d.scan.redo = false
			d.scan.step = stateBeginValue
		}

		return
	}

	d.scan.step = stateBeginValue

	op := d.scanWhile(scanSkipSpace)

	switch op {
	default:
		d.error(errPhase)

	case scanBeginPath:
		d.path(v)

	case scanBeginLiteral:
		d.path(v)

	case scanEnd:
		return
	}
}

// indirect walks down v allocating pointers as needed,
// until it gets to a non-pointer.
// if it encounters an Unmarshaler, indirect stops and returns that.
// if decodingNull is true, indirect stops at the last pointer so it can be set to nil.
func (d *decodeState) indirect(v reflect.Value, decodingNull bool) (Unmarshaler, encoding.TextUnmarshaler, reflect.Value) {
	// If v is a named type and is addressable,
	// start with its address, so that if the type has pointer methods,
	// we find them.
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
	}
	for {
		// Load value from interface, but only if the result will be
		// usefully addressable.
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Ptr && !e.IsNil() && (!decodingNull || e.Elem().Kind() == reflect.Ptr) {
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Ptr {
			break
		}

		if v.Elem().Kind() != reflect.Ptr && decodingNull && v.CanSet() {
			break
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		if v.Type().NumMethod() > 0 {
			if u, ok := v.Interface().(Unmarshaler); ok {
				return u, nil, reflect.Value{}
			}
			if u, ok := v.Interface().(encoding.TextUnmarshaler); ok {
				return nil, u, reflect.Value{}
			}
		}
		v = v.Elem()
	}
	return nil, nil, v
}

// array consumes an array from d.data[d.off-1:], decoding into the value v.
// the first byte of the array ('[') has been read already.
func (d *decodeState) path(v reflect.Value) {

	// Check for unmarshaler.
	u, ut, pv := d.indirect(v, false)

	if u != nil {
		d.off--
		err := u.UnmarshalPath(d.next())
		if err != nil {
			d.error(err)
		}
		return
	}

	if ut != nil {
		d.saveError(&UnmarshalTypeError{"path", v.Type(), int64(d.off)})
		d.off--
		d.next()
		return
	}

	v = pv

	kind := v.Kind()

	// Check type of target.
	switch kind {
	case reflect.Interface:
		if v.NumMethod() == 0 {
			// Decoding into nil interface?  Switch to non-reflect code.
			v.Set(reflect.ValueOf(d.pathInterface()))
			return
		}
		// Otherwise it's invalid.
		fallthrough
	default:
		d.saveError(&UnmarshalTypeError{"path", v.Type(), int64(d.off)})
		d.off--
		d.next()
		return
	case reflect.String:
		d.off--
		d.literalStore(d.next(), pv)
		return
	case reflect.Struct:
		break
	}

	var bufParts []string

	for {
		// Read value.
		start := d.off - 1
		op := d.scanWhile(scanContinue)
		value := d.data[start : d.off-1]

		if string(value) == "/" {
			continue
		}

		bufParts = append(bufParts, string(value))

		if op >= scanEndPath {
			d.off--
			break
		}
	}

	// Searching field to see if we have a match
	//	var f *field
	fields := cachedTypeFields(v.Type())

	for i := range fields {
		ff := &fields[i]

		if ff.parts != "" {
			index := Index{
				format: ff.parts, length: len(bufParts),
			}

			start, _ := index.Start()
			end, _ := index.End()

			var subv reflect.Value

			subv = v
			for _, i := range ff.index {
				if subv.Kind() == reflect.Ptr {
					if subv.IsNil() {
						subv.Set(reflect.New(subv.Type().Elem()))
					}
					subv = subv.Elem()
				}
				subv = subv.Field(i)
			}

			res := bufParts[start:end]

			d.arrayStore(res, subv)
		}
	}
}

// literalStore decodes a literal stored in item into v.
func (d *decodeState) literalStore(item []byte, v reflect.Value) {

	// Check for unmarshaler.
	if len(item) == 0 {
		//Empty string given
		d.saveError(fmt.Errorf("url: invalid use of ,string struct tag, trying to unmarshal %q into %v", item, v.Type()))
		return
	}
	u, ut, pv := d.indirect(v, false)
	if u != nil {
		err := u.UnmarshalPath(item)
		if err != nil {
			d.error(err)
		}
		return
	}
	if ut != nil {
		err := ut.UnmarshalText(item)
		if err != nil {
			d.error(err)
		}
		return
	}

	v = pv

	switch v.Kind() {
	default:
		d.saveError(&UnmarshalTypeError{"string", v.Type(), int64(d.off)})
	case reflect.Slice:
		if v.Type().Elem().Kind() != reflect.Uint8 {
			d.saveError(&UnmarshalTypeError{"string", v.Type(), int64(d.off)})
			break
		}
		b := make([]byte, base64.StdEncoding.DecodedLen(len(item)))
		n, err := base64.StdEncoding.Decode(b, item)
		if err != nil {
			d.saveError(err)
			break
		}
		v.SetBytes(b[:n])
	case reflect.String:
		v.SetString(string(item))
	case reflect.Interface:
		if v.NumMethod() == 0 {
			v.Set(reflect.ValueOf(string(item)))
		} else {
			d.saveError(&UnmarshalTypeError{"string", v.Type(), int64(d.off)})
		}
	}
}

// arrayStore decodes an array stored in item into v.
func (d *decodeState) arrayStore(item []string, v reflect.Value) {

	u, _, pv := d.indirect(v, false)

	if u != nil {
		err := u.UnmarshalPath([]byte(strings.Join(item, "/")))
		if err != nil {
			d.error(err)
		}
		return
	}

	v = pv

	switch v.Kind() {
	default:
		d.saveError(&UnmarshalTypeError{"string", v.Type(), int64(d.off)})
	case reflect.Slice:
		v.Set(reflect.ValueOf([]string(item)))
	case reflect.String:
		v.SetString(strings.Join(item, "/"))
	case reflect.Interface:
		if v.NumMethod() == 0 {
			v.Set(reflect.ValueOf([]string(item)))
		} else {
			d.saveError(&UnmarshalTypeError{"string", v.Type(), int64(d.off)})
		}
	}
}

// The xxxInterface routines build up a value to be stored
// in an empty interface.  They are not strictly necessary,
// but they avoid the weight of reflection in this common case.

// valueInterface is like value but returns interface{}
func (d *decodeState) valueInterface() interface{} {
	switch d.scanWhile(scanSkipSpace) {
	default:
		d.error(errPhase)
		panic("unreachable")
	case scanBeginPath:
		return d.pathInterface()
	}
}

// arrayInterface is like array but returns []interface{}.
func (d *decodeState) pathInterface() []interface{} {
	var v = make([]interface{}, 0)
	for {
		// Look ahead for ] - can only happen on first iteration.
		op := d.scanWhile(scanSkipSpace)
		if op == scanEndPath {
			break
		}

		// Back up so d.value can have the byte we just read.
		d.off--
		d.scan.undo(op)

		v = append(v, d.valueInterface())

		// Next token must be , or ].
		op = d.scanWhile(scanSkipSpace)
		if op == scanEndPath {
			break
		}
		if op != scanBeginPath {
			d.error(errPhase)
		}
	}
	return v
}

// getu4 decodes \uXXXX from the beginning of s, returning the hex value,
// or it returns -1.
func getu4(s []byte) rune {
	if len(s) < 6 || s[0] != '\\' || s[1] != 'u' {
		return -1
	}
	r, err := strconv.ParseUint(string(s[2:6]), 16, 64)
	if err != nil {
		return -1
	}
	return rune(r)
}

// Op defines an mathematical operation
type Op struct {
	Func func(int, int) int
	Next string
}

var ops = map[string]Op{
	"+": {func(x1 int, x2 int) int {
		return x1 + x2
	}, "-"},
	"-": {func(x1 int, x2 int) int {
		return x1 - x2
	}, ""},
}

func eval(s string, opStr string) (int, error) {

	op := ops[opStr]

	var num1 int

	stack := strings.Split(s, opStr)

	if op.Next == "" {
		token := stack[0]
		num1_64, _ := strconv.ParseInt(token, 10, 16)
		num1 = int(num1_64)
	} else {
		num1, _ = eval(stack[0], op.Next)
	}

	if len(stack) == 1 {
		return num1, nil
	}

	num2, _ := eval(strings.Join(stack[1:], opStr), opStr)
	return op.Func(num1, num2), nil
}

// An Index of the buffered parts of the path
type Index struct {
	format string
	length int
}

// Start value of the buffered parts
func (i Index) Start() (int, error) {
	indexes := strings.Split(string(i.format), ":")
	index := indexes[0]

	return i.eval(index)
}

// End value of the buffered parts
func (i Index) End() (int, error) {
	indexes := strings.Split(string(i.format), ":")
	index := indexes[len(indexes)-1]

	return i.eval(index)
}

func (i Index) eval(s string) (int, error) {
	s = strings.Replace(s, "first", string(0), -1)
	s = strings.Replace(s, "last", strconv.Itoa(i.length), -1)

	return eval(s, "+")
}
