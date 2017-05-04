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
 *
 * Package pydiows implements a WebSocket server by
 * piping its input and output through the WebSocket
 * connection.
 */
package pydiows

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/gorilla/websocket"
	"github.com/mholt/caddy/caddyhttp/httpserver"
	pydhttp "github.com/pydio/pydio-booster/http"
	"github.com/pydio/pydio-booster/io"
	"github.com/pydio/pydio-booster/server/middleware/pydiomiddleware"
	pydiows "github.com/pydio/pydio-booster/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 1024 * 1024 * 10 // 10 MB default.
)

type (
	// Handler structure
	Handler struct {
		Next    httpserver.Handler
		Sockets []Config
		Pre     []pydiomiddleware.Rule
	}

	// Config holds the configuration for a single websocket
	// endpoint which may serve multiple websocket connections.
	Config struct {
		Path string
	}

	// UserResponse from the server
	UserResponse struct {
		User pydio.User `xml:"user"`
	}
)

// ServeHTTP converts the HTTP request to a WebSocket connection and serves it up.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {

	for _, sockconfig := range h.Sockets {
		if httpserver.Path(r.URL.Path).Matches(sockconfig.Path) {

			res := errHandle(r, handle(w, r, &sockconfig))

			if res.Err != nil {
				logger.Errorln("PydioWS returns an error : ", res.Err)

				return http.StatusUnauthorized, res.Err
			}

			return http.StatusOK, nil
		}
	}

	// Didn't match a websocket path, so pass-thru
	return h.Next.ServeHTTP(w, r)
}

func errHandle(r *http.Request, f func() *pydhttp.Status) *pydhttp.Status {

	ctx := r.Context()

	c := make(chan *pydhttp.Status, 1)

	if err := ctx.Err(); err != nil {
		return pydhttp.NewStatusErr(http.StatusInternalServerError, err)
	}

	go func() { c <- f() }()

	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != nil {
			return pydhttp.NewStatusErr(http.StatusInternalServerError, err)
		}
	case res := <-c:
		return res
	}

	return pydhttp.NewStatusOK(r)
}

// serveWS is used for setting and upgrading the HTTP connection to a websocket connection.
// It also spawns the child process that is associated with matched HTTP path/url.
func handle(w http.ResponseWriter, r *http.Request, config *Config) func() *pydhttp.Status {

	return func() *pydhttp.Status {
		logger.Infoln("PydioWS : handler START")

		var conn *websocket.Conn
		var err error

		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		}

		ctx := r.Context()

		// Retrieving the node
		logger.Debugln("PydioWS : get context user")
		userResponse := UserResponse{}
		if err = getValue(ctx, "user", &userResponse); err != nil {
			return pydhttp.NewStatusErr(http.StatusInternalServerError, err)
		}
		user := userResponse.User
		logger.Debugln("PydioWS : got context user ", user)

		logger.Debugln("PydioWS : Upgrader")
		conn, err = upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Errorln("Error upgrading ", err)
			return pydhttp.NewStatusErr(http.StatusInternalServerError, err)
		}

		defer conn.Close()
		logger.Infoln("PydioWS : Upgraded")

		// Request Read / Writer
		reqr, reqw := io.Pipe()

		// Response Read / Writer
		respr, respw := io.Pipe()
		defer respw.Close()

		// Creating Websocket Connection
		connection, err := pydiows.NewConnection(&user, reqr, respw)
		if err != nil {
			return pydhttp.NewStatusErr(http.StatusInternalServerError, err)
		}

		done := make(chan struct{})
		go pumpStdout(conn, respr, done)
		pumpStdin(conn, reqw)

		reqw.Close() // close stdin to end the process

		logger.Debugln("Closed the websocket")

		select {
		case <-connection.ExitChan:
		case <-time.After(time.Second):
			<-done
		}

		logger.Infoln("PydioWS : handler END")

		return nil
	}
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

	dec := xml.NewDecoder(strings.NewReader(data))
	if err := dec.Decode(&value); err != nil {
		logger.Errorf("value for %s : %v", key, err)
		return err
	}

	return nil
}

// pumpStdin handles reading data from the websocket connection and writing
// it to stdin of the process.
func pumpStdin(conn *websocket.Conn, stdin io.WriteCloser) {
	// Setup our connection's websocket ping/pong handlers from our const values.
	defer conn.Close()
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		message = append(message, '\n')
		if _, err := stdin.Write(message); err != nil {
			break
		}
	}
}

// pumpStdout handles reading data from stdout of the process and writing
// it to websocket connection.
func pumpStdout(conn *websocket.Conn, stdout io.Reader, done chan struct{}) {
	go pinger(conn, done)
	defer func() {
		conn.Close()
		close(done) // make sure to close the pinger when we are done.
	}()

	s := bufio.NewScanner(stdout)
	for s.Scan() {
		conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteMessage(websocket.TextMessage, bytes.TrimSpace(s.Bytes())); err != nil {
			break
		}
	}
	if s.Err() != nil {
		conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, s.Err().Error()), time.Time{})
	}
}

// pinger simulates the websocket to keep it alive with ping messages.
func pinger(conn *websocket.Conn, done chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for { // blocking loop with select to wait for stimulation.
		select {
		case <-ticker.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
				conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, err.Error()), time.Time{})
				return
			}
		case <-done:
			return // clean up this routine.
		}
	}
}
