/*Package com controls the communication layer of the Pydio app
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
package com

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/nsqio/nsq/nsqd"
	pydioconf "github.com/pydio/pydio-booster/conf"
	"github.com/pydio/pydio-booster/log"
)

// Message standard structure for communication
type Message struct {
	Topic   string
	Content []byte
}

type instance struct {
	doneChan chan bool
	exitChan chan bool
	running  bool
}

var (
	opts   *nsqd.Options
	unique instance
)

// NewCom Consumer
func NewCom(conf *pydioconf.NsqConf) error {

	// making sure we re not in the middle of an exit
	if (unique != instance{}) {
		<-unique.exitChan
	}

	if IsRunning() {
		return errors.New("NSQ already running")
	}

	opts = getOptions()

	tcpPort := conf.Port
	tcpHost := conf.Host
	tcpPort = GetNextAvailablePort(tcpPort)
	opts.TCPAddress = tcpHost + ":" + strconv.Itoa(tcpPort)
	httpPort := GetNextAvailablePort(tcpPort + 1)
	opts.HTTPAddress = tcpHost + ":" + strconv.Itoa(httpPort)

	opts.Logger = log.New(log.INFO, "[nsqd] ", log.Ldate|log.Ltime|log.Lmicroseconds)

	//opts.NSQLookupdTCPAddresses = []string{"0.0.0.0:4160"}

	nsqd := nsqd.New(opts)

	log.Infof("[com] Starting NSQ on port %d and %d\n", tcpPort, httpPort)

	nsqd.Main()

	unique.running = true
	unique.doneChan = make(chan (bool))
	unique.exitChan = make(chan (bool))

	// Make sure we exit the nsqd
	go func() {
		// wait until we are told to continue and exit
		<-unique.doneChan

		nsqd.Exit()

		unique.running = false
		unique.exitChan <- true
	}()

	return nil
}

// Close the nsq
func Close() {
	WaitForConsumers()
	unique.doneChan <- true
}

// IsRunning nsq ?
func IsRunning() bool {
	return unique != (instance{}) && unique.running
}

// GetNextAvailablePort for the
func GetNextAvailablePort(port int) int {
	if !isTCPPortAvailable(port) {
		return GetNextAvailablePort(port + 1)
	}
	return port
}

func isTCPPortAvailable(port int) bool {
	conn, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func getOptions() *nsqd.Options {
	if opts != nil {
		return opts
	}

	return nsqd.NewOptions()
}
