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
package pydioadmin

import (
	"net/http"

	"github.com/mholt/caddy/caddyhttp/httpserver"
	"github.com/pydio/pydio-booster/conf"
	"gopkg.in/square/go-jose.v1/json"
)

// Handler for the pydio middleware
type Handler struct {
	Next  httpserver.Handler
	Rules []Rule
}

// VersionResponse structure
type VersionResponse struct {
	VersionString string
	VersionDate   string
}

// Rule for the Handler
type Rule struct {
	Path string
}

// ServerHTTP Requests for uploading files to the server
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {

	switch r.Method {
	case http.MethodGet, http.MethodPost:
		for _, rule := range h.Rules {
			if httpserver.Path(r.URL.Path).Matches(rule.Path) {
				return handle(w, r)
			}
		}
	}

	return h.Next.ServeHTTP(w, r)
}

func handle(w http.ResponseWriter, r *http.Request) (int, error) {

	w.Header().Add("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	response := &VersionResponse{}
	response.VersionString = conf.VersionString
	response.VersionDate = conf.VersionDate
	err := encoder.Encode(response)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}
