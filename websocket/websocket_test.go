// Package websocket contains the code to create and handle a Pydio websocket connection
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
package websocket

import (
	"bufio"
	"io"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/pydio/pydio-booster/com"
	pydioconf "github.com/pydio/pydio-booster/conf"
	pydio "github.com/pydio/pydio-booster/io"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	fakeUser    *pydio.User
	fakeToken   string
	fakeMessage com.Message

	secret string
)

func init() {

	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)

	secret = "TestingSecret"

	com.NewCom(&pydioconf.NsqConf{
		Host: "localhost",
		Port: 4150,
	})

	fakeUser = &pydio.User{
		ID:        "test",
		GroupPath: "test",
		Repos: []pydio.Repo{
			{ID: "test"},
		},
	}

	fakeMessage = com.Message{
		Topic:   "im",
		Content: []byte("{\"REPO_ID\":\"test\", \"CONTENT\":\"This is a simple test\"}"),
	}

	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)

	fakeToken = generateTmpJWT()
}

// Generates a new JWT Token.
func generateTmpJWT() string {
	token, _ := fakeUser.JWT("secret", 24)

	return token
}

func waitForResponse(reader io.Reader, writer io.Writer) string {

	scanner := bufio.NewScanner(reader)

	// setting timeout
	go func() {
		<-time.After(100 * time.Millisecond)

		writer.Write([]byte("TIMEOUT\n"))
	}()

	for scanner.Scan() {
		return scanner.Text()
	}

	return "TIMEOUT"
}

func TestSuccess(t *testing.T) {

	com.NewProducer()
	defer com.StopProducer()

	Convey("Testing a websocket connection", t, func() {
		reqr, reqw := io.Pipe()
		defer reqw.Close()

		respr, respw := io.Pipe()
		defer respw.Close()

		connection, err := NewConnection(fakeUser, reqr, respw)
		So(err, ShouldBeNil)

		// Registering to a wrong repo
		reqw.Write([]byte("register:evil\n"))
		So(connection.Repo, ShouldBeNil)

		// Registering
		reqw.Write([]byte("register:test\n"))
		So(connection.Repo, ShouldNotBeNil)
		So(connection.Repo.ID, ShouldEqual, "test")

		// Unregistering
		reqw.Write([]byte("unregister\n"))
		So(connection.Repo, ShouldBeNil)

		// Publishing a message while not registered
		com.Publish(com.Message{
			Topic:   "im",
			Content: []byte("{\"REPO_ID\":\"test\", \"CONTENT\":\"This is a simple test\"}"),
		})
		So(waitForResponse(respr, respw), ShouldEqual, "TIMEOUT")

		// Publishing a message for another group while registered to the repo
		com.Publish(com.Message{
			Topic:   "im",
			Content: []byte("{\"GROUP_PATH\": \"evil\", \"REPO_ID\":\"test\", \"CONTENT\":\"This is a simple test\"}"),
		})
		So(waitForResponse(respr, respw), ShouldEqual, "TIMEOUT")

		// Publishing a message for another user while registered to the repo
		com.Publish(com.Message{
			Topic:   "im",
			Content: []byte("{\"USER_ID\": \"evil\", \"REPO_ID\":\"test\", \"CONTENT\":\"This is a simple test\"}"),
		})
		So(waitForResponse(respr, respw), ShouldEqual, "TIMEOUT")

		// Registering
		reqw.Write([]byte("register:test\n"))
		So(connection.Repo, ShouldNotBeNil)
		So(connection.Repo.ID, ShouldEqual, "test")

		// Publishing a message for another group while registered to the repo
		com.Publish(com.Message{
			Topic:   "im",
			Content: []byte("{\"REPO_ID\":\"test\", \"CONTENT\":\"This is a simple test\"}"),
		})
		So(waitForResponse(respr, respw), ShouldEqual, "This is a simple test")

		// Publishing a message for another group while registered to the repo
		com.Publish(com.Message{
			Topic:   "im",
			Content: []byte("{\"REPO_ID\":\"test\", \"USER_ID\":\"test\", \"CONTENT\":\"This is a simple test\"}"),
		})
		So(waitForResponse(respr, respw), ShouldEqual, "This is a simple test")

		// Publishing a message for another group while registered to the repo
		com.Publish(com.Message{
			Topic:   "im",
			Content: []byte("{\"REPO_ID\":\"test\", \"GROUP_PATH\":\"test\", \"CONTENT\":\"This is a simple test\"}"),
		})
		So(waitForResponse(respr, respw), ShouldEqual, "This is a simple test")
	})
}

func BenchmarkPublish(b *testing.B) {

	com.NewProducer()
	defer com.StopProducer()

	reqr, reqw := io.Pipe()
	defer reqw.Close()

	_, respw := io.Pipe()
	defer respw.Close()

	connection, _ := NewConnection(fakeUser, reqr, respw)

	Convey("Testing a websocket connection", b, func() {

		reqw.Write([]byte("register:test\n"))
		So(connection.Repo, ShouldNotBeNil)
		So(connection.Repo.ID, ShouldEqual, "test")

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				com.Publish(fakeMessage)
			}
		})
	})
}

func BenchmarkConsume(b *testing.B) {
	Convey("Testing a websocket connection receiver", b, func() {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				reqr, reqw := io.Pipe()
				defer reqw.Close()

				NewConnection(fakeUser, reqr, os.Stdout)
				reqw.Write([]byte("register:test\n"))

			}
		})
	})
}
