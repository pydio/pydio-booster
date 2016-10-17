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
package pydiomiddleware

import (
	"net/http"
	"net/url"
	"regexp"

	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
	"github.com/pydio/pydio-booster/http"
)

// Parse the middleware rules
func Parse(c *caddy.Controller, path string, middlewares ...string) (rules map[string][]Rule, err error) {

	rules = make(map[string][]Rule)

	for {
		for _, middleware := range middlewares {
			if c.Val() == middleware {

				rule, err := parseRule(c)
				rule.Path = path

				if err != nil {
					return nil, err
				}

				rules[middleware] = append(rules[middleware], rule)
			}
		}

		if !c.Next() {
			break
		}
	}

	return rules, nil
}

func parseRule(c *caddy.Controller) (Rule, error) {
	var rule Rule

	var matcher httpserver.RequestMatcher
	var err error

	// Integrate request matcher for 'if' conditions.
	matcher, err = httpserver.SetupIfMatcher(c)
	if err != nil {
		return rule, err
	}

	for c.NextBlock() {

		if httpserver.IfMatcherKeyword(c) {
			continue
		}

		switch c.Val() {
		case "pattern":
			if !c.NextArg() {
				return rule, nil
			}
			pattern := c.Val()

			r, err := regexp.Compile(pattern)

			if err != nil {
				return rule, c.ArgErr()
			}

			rule.Regexp = r
		case "url":
			urlString := c.RemainingArgs()
			url, err := url.Parse(urlString[0])

			if err != nil {
				return rule, c.ArgErr()
			}

			rule.URL = *url

		case "query":
			queryArgs := c.RemainingArgs()

			rule.QueryArgs = make(map[string][]string)

			if len(queryArgs) < 2 {
				rule.QueryMatchers = append(rule.QueryMatchers, queryArgs[0])
			}
		case "cookie":
			cookies := c.RemainingArgs()

			if len(cookies) < 2 {
				rule.CookieMatchers = append(rule.CookieMatchers, pydhttp.CookieMatcher(cookies[0]))
			} else {
				rule.Cookies = append(rule.Cookies, &http.Cookie{Name: cookies[0], Value: cookies[1]})
			}

		case "header":
			headers := c.RemainingArgs()

			if len(headers) < 2 {
				rule.HeaderMatchers = append(rule.HeaderMatchers, headers[0])
			} else {
				rule.Headers = append(rule.Headers, [2]string{headers[0], headers[1]})
			}

		case "type":
			queryType := c.RemainingArgs()
			rule.QueryType = queryType[0]

		case "out":
			out := c.RemainingArgs()
			rule.Out = out[0]
		}
	}

	rule.Matcher = matcher

	return rule, nil
}
