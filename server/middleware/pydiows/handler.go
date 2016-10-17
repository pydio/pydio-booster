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
	"io"
	"log"
	"net/http"
	"time"

	"golang.org/x/net/context"

	"github.com/gorilla/websocket"
	"github.com/mholt/caddy/caddyhttp/httpserver"
	"github.com/pydio/pydio-booster/http"
	"github.com/pydio/pydio-booster/io"
	"github.com/pydio/pydio-booster/server/middleware/pydiomiddleware"
	pydiows "github.com/pydio/pydio-booster/websocket"
	"github.com/pydio/go/io"
	"github.com/pydio/go/http"
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
)

// ServeHTTP converts the HTTP request to a WebSocket connection and serves it up.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {

	for _, sockconfig := range h.Sockets {
		if httpserver.Path(r.URL.Path).Matches(sockconfig.Path) {
			ctx := r.Context()

			err := errHandle(ctx, handle(w, r, &sockconfig))

			if err != nil {
				log.Println("PydioWS returns an error : ", err)

				return http.StatusUnauthorized, err
			}

			return http.StatusOK, nil
		}
	}

	// Didn't match a websocket path, so pass-thru
	return h.Next.ServeHTTP(w, r)
}

func errHandle(ctx context.Context, f func() error) error {

	c := make(chan error, 1)

	if err := ctx.Err(); err != nil {
		return err
	}

	go func() { c <- f() }()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-c:
		return err
	}
}

// serveWS is used for setting and upgrading the HTTP connection to a websocket connection.
// It also spawns the child process that is associated with matched HTTP path/url.
func handle(w http.ResponseWriter, r *http.Request, config *Config) func() error {
	return func() error {
		log.Println("PydioWS : handler START")

		var conn *websocket.Conn
		var err error

		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		}

		ctx := r.Context()

		// Retrieving the node
		log.Println("PydioWS : get context user")
		var user = &pydio.User{}

		if err = pydhttp.FromContext(ctx, "user", user); err != nil {
			log.Println("Could not decode to User ", err)
			return err
		}
		log.Println("PydioWS : got context user ", user)

		log.Println("PydioWS : Upgrader")
		conn, err = upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Error upgrading ", err)
			return err
		}

		defer conn.Close()
		log.Println("PydioWS : Upgraded")

		// Request Read / Writer
		reqr, reqw := io.Pipe()

		// Response Read / Writer
		respr, respw := io.Pipe()
		defer respw.Close()

		// Creating Websocket Connection
		connection, err := pydiows.NewConnection(user, reqr, respw)
		if err != nil {
			return err
		}

		done := make(chan struct{})
		go pumpStdout(conn, respr, done)
		pumpStdin(conn, reqw)

		reqw.Close() // close stdin to end the process

		log.Println("We're here")

		select {
		case <-connection.ExitChan:
		case <-time.After(time.Second):
			<-done
		}

		log.Println("PydioWS : handler END")

		return nil
	}
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
