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
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/mholt/caddy/caddyhttp/httpserver"

	pydhttp "github.com/pydio/pydio-booster/http"
	pydio "github.com/pydio/pydio-booster/io"
	pydioworker "github.com/pydio/pydio-booster/worker"
)

// Handler for the pydio middleware
type Handler struct {
	Next       httpserver.Handler
	Rules      []Rule
	Dispatcher *pydioworker.Dispatcher
}

// ServerHTTP Requests for uploading files to the server
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {

	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut:
		for _, rule := range h.Rules {
			if !rule.Matcher.Match(r) {
				log.Println("[ERROR:MW] Not a match")
				continue
			}

			if httpserver.Path(r.URL.Path).Matches(rule.Path) {

				parent := r.Context()

				ctx, cancel := context.WithCancel(parent)

				r = r.WithContext(ctx)

				ctx, statusCode, err := handle(&rule, h.Dispatcher, w, r, cancel)

				if err != nil || statusCode != 0 {
					log.Println("got a status code returned ", statusCode, err)
					return statusCode, err
				}

				r = r.WithContext(ctx)

				continue
			}
		}
	}

	return h.Next.ServeHTTP(w, r)
}

func handle(rule *Rule, d *pydioworker.Dispatcher, w http.ResponseWriter, r *http.Request, cancel func()) (context.Context, int, error) {

	log.Println("[REQ:MW] START")

	start := time.Now()

	defer func() {
		elapsed := time.Since(start)
		log.Printf("[REQ:MW] END - took %s", elapsed)
	}()

	/**********************************************
	-- Retrieving parameters from context
	***********************************************/
	ctx := r.Context()
	replacer := httpserver.NewReplacer(r, nil, "")

	/**********************************************
	-- Defining the context variables to be added
	***********************************************/
	out := pydhttp.NewContextValue()
	encoder := json.NewEncoder(out)

	ctx = context.WithValue(ctx, rule.Out, out)

	// Fill in cookies
	values := &url.Values{}
	url := url.URL{}
	if rule.URL == url {
		url = *r.URL
	} else {
		url = rule.URL
	}

	if rule.Regexp != nil {
		matches := rule.Regexp.FindStringSubmatch(r.URL.Path)

		for i := 1; i < len(matches); i++ {
			replacer.Set(fmt.Sprint(i), matches[i])
		}

		log.Println("[INFO:MW] We have a replacer here ", replacer, rule.Regexp, matches)
	}

	// Matching potential headers to forward
	headers := rule.Headers
	for key := range r.Header {
		for _, matcher := range rule.HeaderMatchers {
			if matcher != "*" && matcher != key {
				continue
			}

			headers = append(rule.Headers, [2]string{key, r.Header.Get(key)})
		}
	}

	// Matching potential query arguments to forward
	for key, value := range url.Query() {
		values.Set(key, value[0])
	}
	for key, value := range r.URL.Query() {
		for _, matcher := range rule.QueryMatchers {
			if matcher != "*" && matcher != key {
				continue
			}

			values.Set(key, value[0])
		}
	}
	url.RawQuery = values.Encode()

	// Matching potential cookies to forward
	cookies := rule.Cookies
	for _, cookie := range r.Cookies() {
		for _, matcher := range rule.CookieMatchers {
			if !matcher.Match(cookie) {
				continue
			}

			// This is a match
			cookies = append(cookies, cookie)
		}
	}

	log.Printf("[INFO:MW] Working with %s", url.String())
	var job pydioworker.Job
	var err error
	switch rule.QueryType {
	case "node":
		job, err = NewNodeJob(url, ctx, replacer, *encoder, w, out.Close, cancel)
	case "auth":
		job, err = NewAuthJob(url, ctx, replacer, *encoder, w, out.Close, cancel)
	case "request":
		job, err = NewRequestJob(url, headers, cookies, rule.Out, ctx, replacer, *encoder, w, out.Close, cancel)
	}

	if err != nil {
		return nil, http.StatusUnauthorized, err
	}

	if rule.Out == "body" {
		job.Do()
		return ctx, http.StatusOK, nil
	}

	d.Add(job)

	return ctx, 0, nil
}

func getContextNode(ctx context.Context) (*pydio.Node, error) {

	node := &pydio.Node{}
	if err := pydhttp.FromContext(ctx, "node", node); err != nil {
		log.Println("PydioPre : Could not decode to Node ", err)
		return nil, err
	}

	return node, nil
}

func getContextAuthParams(ctx context.Context, url string) (auth *pydhttp.Auth, err error) {

	// Retrieving auth from headers
	auth = &pydhttp.Auth{}
	if err = pydhttp.FromContext(ctx, "auth", auth); err != nil {
		log.Println("PydioPre : Could not decode to auth")
	}

	if auth.Token == "" {
		// Retrieving token from headers
		var token = &pydhttp.Token{}
		if err = pydhttp.FromContext(ctx, "token", token); err != nil {
			log.Println("PydioPre : Could not decode to token ", err)
			return nil, err
		}

		// Building Query
		args := token.GetQueryArgs(url)

		auth.Hash = args.Hash
		auth.Token = args.Token
	}

	return
}

type (
	// Rule for the Handler
	Rule struct {
		Path           string
		URL            url.URL
		Regexp         *regexp.Regexp
		CookieMatchers []pydhttp.CookieMatcher
		Cookies        []*http.Cookie
		QueryMatchers  []string
		QueryArgs      map[string][]string
		HeaderMatchers []string
		Headers        [][2]string
		QueryType      string
		Out            string

		Matcher httpserver.RequestMatcher
	}

	// PathQuery structure
	PathQuery struct {
		Action string     `path:",0:1"`
		Node   pydio.Node `path:",1:last"`
	}

	query interface{}

	// UserQuery xml
	UserQuery struct {
		User pydio.User `xml:"user" json:"u"`
	}

	// OptionsQuery json
	OptionsQuery struct {
		pydio.Options
	}

	// TokenQuery json
	TokenQuery pydhttp.Token

	// ProxyQuery json
	ProxyQuery struct {
		Key         string `json:"key"`
		ClientID    string `json:"client"`
		SourceURL   string `json:"source_url"`
		ShortenType string `json:"shorten_type"`
		PublicURL   string `json:"public_url"`
	}

	// ClientQuery json
	ClientQuery struct {
		ID string
	}
)
