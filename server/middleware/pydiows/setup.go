// Package pydiows contains the logic for the pydiows caddy directive
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
package pydiows

import (
	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
	"github.com/pydio/pydio-booster/server/middleware/pydiomiddleware"

	pydiolog "github.com/pydio/pydio-booster/log"
	pydioworker "github.com/pydio/pydio-booster/worker"
)

var logger *pydiolog.Logger

func init() {
	caddy.RegisterPlugin("pydiows", caddy.Plugin{
		ServerType: "http",
		Action:     setup,
	})

	logger = pydiolog.New(pydiolog.GetLevel(), "[pydiows] ", pydiolog.Ldate|pydiolog.Ltime|pydiolog.Lmicroseconds)
}

// Setup the Pydio websocket middleware instance.
func setup(c *caddy.Controller) error {

	cfg := httpserver.GetConfig(c)

	websocks, middlewareRules, err := webSocketParse(c)
	if err != nil {
		return err
	}

	dispatcher := pydioworker.NewDispatcher(900)
	dispatcher.Run()

	// Pre Middlewares
	cfg.AddMiddleware(func(next httpserver.Handler) httpserver.Handler {
		return &pydiomiddleware.Handler{
			Next:       next,
			Rules:      middlewareRules["pre"],
			Dispatcher: dispatcher,
		}
	})

	cfg.AddMiddleware(func(next httpserver.Handler) httpserver.Handler {
		return &Handler{
			Next:    next,
			Sockets: websocks,
		}
	})

	return nil
}

func webSocketParse(c *caddy.Controller) (websocks []Config, middlewareRules map[string][]pydiomiddleware.Rule, err error) {

	for c.Next() {
		var path string

		// Path or command; not sure which yet
		if !c.NextArg() {
			return websocks, nil, c.ArgErr()
		}

		path = c.Val()

		websocks = append(websocks, Config{
			Path: path,
		})

		if c.NextBlock() {
			middlewareRules, _ = pydiomiddleware.Parse(c, path, "pre")
		}
	}

	return websocks, middlewareRules, nil
}
