// Package pydiomiddleware contains the logic for a middleware directive (repetitive task done for a Pydio request)
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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
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
				logger.Errorln("Not a match")
				continue
			}

			if httpserver.Path(r.URL.Path).Matches(rule.Path) {

				parent := r.Context()

				ctx, cancel := context.WithCancel(parent)

				r = r.WithContext(ctx)

				ctx, statusCode, err := handle(&rule, h.Dispatcher, w, r, cancel)

				if err != nil || statusCode != 0 {
					if err != nil {
						logger.Errorln("got an error returned ", statusCode, err)
					} else {
						logger.Infoln("got a status code ", statusCode)
					}
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

	logger.Infoln("START")

	start := time.Now()

	defer func() {
		elapsed := time.Since(start)
		logger.Infoln("END - took %s", elapsed)
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
	encoder := rule.EncoderFunc(out)

	ctx = context.WithValue(ctx, rule.Out.Name, out)

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

		logger.Debugln("replacer : ", replacer, rule.Regexp, matches)
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

			if len(value) > 1 {
				for _, val := range value {

					logger.Debugln("Adding ", key, val)
					values.Add(key, val)
				}
			} else {
				values.Set(key, value[0])
			}

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

	var job pydioworker.Job
	var err error
	switch rule.QueryType {
	case "node":
		job, err = NewNodeJob(ctx, url, encoder, out.Close, cancel)
	case "auth":
		job, err = NewAuthJob(ctx, url, encoder, out.Close, cancel)
	case "request":
		job, err = NewRequestJob(ctx, url, headers, cookies, rule.Out, replacer, encoder, w, out.Close, cancel)
	}

	if err != nil {
		return nil, http.StatusUnauthorized, err
	}

	if rule.Out.Name == "body" {
		job.Do()
		return ctx, http.StatusOK, nil
	}

	d.Add(job)

	return ctx, 0, nil
}

func getContextNode(ctx context.Context) (*pydio.Node, error) {

	var node *pydio.Node
	if err := getValue(ctx, "node", &node); err != nil {
		logger.Errorln("Could not decode to Node ", err)
		return nil, err
	}

	return node, nil
}

func getContextAuthParams(ctx context.Context, url string) (*pydhttp.Auth, error) {

	var err error

	// Retrieving auth from headers
	var auth *pydhttp.Auth
	if err = getValue(ctx, "auth", &auth); err != nil {
		logger.Errorln("Could not decode to auth")
	}

	if auth == nil {

		// Retrieving token from headers
		var token *pydhttp.Token
		if err = getValue(ctx, "token", &token); err != nil {
			logger.Errorln("Could not decode to token ", err)
			return nil, err
		}

		// Building Query
		args := token.GetQueryArgs(url)

		auth = &pydhttp.Auth{
			Hash:  args.Hash,
			Token: args.Token,
		}
	}

	return auth, err
}

// asynchronously retrieve values sitting in the context
func getValue(ctx context.Context, key string, value interface{}) error {

	// var node *pydio.Node
	var buf bytes.Buffer

	if err := pydhttp.FromContext(ctx, key, &buf); err != nil {
		return err
	}

	data := buf.String()
	if unquoted, err := strconv.Unquote(strings.Trim(data, "\n")); err == nil {
		data = unquoted
	}

	dec := json.NewDecoder(strings.NewReader(data))
	if err := dec.Decode(&value); err != nil {
		logger.Errorf("value for %s : %v", key, err)
		return err
	}

	return nil
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
		Out            Out
		EncoderFunc    EncoderFunc

		Matcher httpserver.RequestMatcher
	}

	// Out values
	Out struct {
		Name string
		Key  string
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
		Key              string `json:"key"`
		ClientID         string `json:"client"`
		Hash             string `json:"hash"`
		Base             string `json:"base"`
		MainEndpoint     string `json:"main_endpoint"`
		DownloadEndpoint string `json:"dl_endpoint"`
		ShortenType      string `json:"shorten_type"`
		PublicURL        string `json:"public_url"`
	}

	// ClientQuery json
	ClientQuery struct {
		ID string
	}
)
