// Package pydhttp contains all http related work
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
package pydhttp

// Result structure
import (
	"net/http"

	"golang.org/x/net/context"
)

// Status  structure
type Status struct {
	Context    context.Context
	StatusCode int
	Err        error
}

// NewStatusOK response
func NewStatusOK(r *http.Request, context ...context.Context) *Status {

	if len(context) == 1 {
		return &Status{
			Context:    context[0],
			StatusCode: http.StatusOK,
			Err:        nil,
		}
	}

	return &Status{
		Context:    r.Context(),
		StatusCode: http.StatusOK,
		Err:        nil,
	}
}

// NewStatusErr response
func NewStatusErr(code int, err error) *Status {
	return &Status{
		Context:    nil,
		StatusCode: code,
		Err:        err,
	}
}
