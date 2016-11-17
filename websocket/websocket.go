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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/nsqio/go-nsq"
	"github.com/nu7hatch/gouuid"
	"github.com/pydio/pydio-booster/com"
	"github.com/pydio/pydio-booster/io"

	pydiolog "github.com/pydio/pydio-booster/log"
)

// Connection details of a websocket
type Connection struct {
	uniqueID string

	User *pydio.User
	Repo *pydio.Repo

	ExitChan chan (error)

	Incoming io.Reader
	Outgoing io.Writer

	Logger *pydiolog.Logger
}

// PydioInstantMessage format
type PydioInstantMessage struct {
	UserID     string `json:"USER_ID"`
	GroupPath  string `json:"GROUP_PATH"`
	RepoID     string `json:"REPO_ID"`
	XMLContent string `json:"CONTENT"`
}

// NewConnection via a websocket
func NewConnection(u *pydio.User, incoming io.Reader, outgoing io.Writer) (*Connection, error) {

	// Creating a Unique ID for the connection
	u4, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	rc, ok := incoming.(io.ReadCloser)
	if !ok && incoming != nil {
		rc = ioutil.NopCloser(incoming)
	}

	wc, ok := outgoing.(io.WriteCloser)
	if !ok {
		return nil, errors.New("Failed to create a Write closer")
	}

	// Creating the channel for messages that the websocket
	exitChan := make(chan (error))

	connection := &Connection{
		uniqueID: u4.String(),
		User:     u,
		ExitChan: exitChan,
		Incoming: rc,
		Outgoing: wc,

		Logger: pydiolog.New(pydiolog.GetLevel(), "[ws] ", pydiolog.Ldate|pydiolog.Ltime|pydiolog.Lmicroseconds),
	}

	// Create the incoming handler
	go func() {
		reader := connection.Incoming

		scanner := bufio.NewScanner(reader)

		for scanner.Scan() {
			text := scanner.Text()

			if strings.Index(text, "register") == 0 {
				repoID := strings.TrimPrefix(text, "register:")

				repo := connection.User.GetRepo(repoID)
				if !repo.IsReadable() {
					// Retrieving a nil value
					connection.Repo = connection.User.GetRepo("")
					connection.Logger.SetPrefix("[ws] ")
					connection.Logger.Infof("Register %s : repo is writeonly", repoID)
				} else {
					connection.Repo = repo
					connection.Logger.SetPrefix(fmt.Sprintf("[ws %s] ", connection))
					connection.Logger.Infof("Register %s %v", repoID, connection.User.Repos)
				}
			} else if strings.Index(text, "unregister") == 0 {
				// Retrieving a nil value
				connection.Repo = connection.User.GetRepo("")
				connection.Logger.SetPrefix("[ws] ")
				connection.Logger.Infof("Unregister", connection)
			}
		}
	}()

	// Create the handler for incoming messages from the back (NSQ messages)
	go func() {
		// Create consumer for User
		c, err := com.NewConsumer("im", u4.String())
		if err != nil {
			connection.ExitChan <- err
			return
		}

		writer := connection.Outgoing

		c.AddHandler(nsq.HandlerFunc(func(m *nsq.Message) error {
			var pm PydioInstantMessage

			body := m.Body
			user := connection.User
			repo := connection.Repo

			if repo == nil {
				connection.Logger.Debugln("Returning - Empty repository")
				return nil
			}

			if repo.IsReadable() {
				connection.Logger.Debugln("Returning - Write Only repository")
				return nil
			}

			err := json.Unmarshal(body, &pm)
			if err != nil {
				connection.Logger.Errorln(err)
				return err
			}

			if pm.RepoID == "*" || repo.ID == pm.RepoID {
				if pm.UserID != "" && pm.UserID != user.ID {
					return nil
				}

				if pm.GroupPath != "" && pm.GroupPath != user.GroupPath {
					return nil
				}

				connection.Logger.Debugf("Writing to websocket : %s", pm.XMLContent)

				writer.Write([]byte(pm.XMLContent + "\n"))
			}

			return nil
		}))

		c.Start()
	}()

	return connection, nil
}

// ResetPrefix for the logger based on arguments
func (c *Connection) String() string {

	str := fmt.Sprintf("%s:%s", c.User.ID, c.User.GroupPath)

	if c.Repo != nil {
		str = fmt.Sprintf("%s@%s", str, c.Repo.ID)
	}

	return str
}
