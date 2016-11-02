// Package pydioupload contains the logic for the pydioupload caddy directive
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
	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"

	pydiolog "github.com/pydio/pydio-booster/log"
	"github.com/pydio/pydio-booster/server/middleware/pydiomiddleware"
	pydioworker "github.com/pydio/pydio-booster/worker"
)

var logger *pydiolog.Logger

func init() {
	caddy.RegisterPlugin("pydioupload", caddy.Plugin{
		ServerType: "http",
		Action:     setup,
	})

	logger = pydiolog.New(pydiolog.GetLevel(), "[pydioupload] ", pydiolog.Ldate|pydiolog.Ltime|pydiolog.Lmicroseconds)
}

// Setup configures a new PydioUpload instance.
func setup(c *caddy.Controller) error {

	cfg := httpserver.GetConfig(c)

	rules, middlewareRules, err := parse(c)
	if err != nil {
		return err
	}

	logger.Debugln("Got middleware Rules ", middlewareRules)

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
			Next:       next,
			Rules:      rules,
			Dispatcher: dispatcher,
		}
	})

	// Post Middlewares
	cfg.AddMiddleware(func(next httpserver.Handler) httpserver.Handler {
		return &pydiomiddleware.Handler{
			Next:       next,
			Rules:      middlewareRules["post"],
			Dispatcher: dispatcher,
		}
	})

	return nil
}

// parses the config from the caddy file
func parse(c *caddy.Controller) (rules []Rule, middlewareRules map[string][]pydiomiddleware.Rule, err error) {

	for c.Next() {
		var rule Rule

		args := c.RemainingArgs()

		switch len(args) {
		case 1:
			rule.Path = args[0]
		}

		if c.NextBlock() {
			middlewareRules, _ = pydiomiddleware.Parse(c, rule.Path, "pre", "post")
		}

		rules = append(rules, rule)
	}

	return
}
