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
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"

	pydio "github.com/pydio/pydio-booster/io"
	"github.com/pydio/pydio-booster/websocket"
)

func init() {
	// Creating and registering the log file
	log.SetOutput(&lumberjack.Logger{
		Filename:   "/tmp/pydio.out",
		MaxSize:    100,
		MaxAge:     14,
		MaxBackups: 10,
	})
	log.SetPrefix("[ws] ")
}

func main() {

	// Get the authorization code
	auth := os.Getenv("HTTP_AUTHORIZATION")
	auth = strings.TrimPrefix(auth, "Bearer ")

	if auth == "" {
		log.Panicln("Empty authorization")
	}

	log.Printf("Bearer :%s\n", auth)

	// Get user information
	user, err := pydio.NewUserFromJWT(auth, "secret")
	if err != nil {
		log.Panicln("Failed to retrieve user: ", err)
	}

	connection, err := websocket.NewConnection(user, os.Stdin, os.Stdout)
	if err != nil {
		log.Panicln("Failed to create connection: ", err)
	}

	log.SetPrefix(fmt.Sprintf("[ws %s]", connection))
	log.Println("Connected")

	// Listening for socket closing messages messages
	for {
		select {
		case err := <-connection.ExitChan:
			log.Println("Error :", err)
		}
	}

	log.Println("Ending connection")
}
