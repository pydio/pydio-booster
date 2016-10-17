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
package com

import (
	"fmt"
	"testing"

	"github.com/nsqio/go-nsq"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/pydio/pydio-booster/conf"
)

func TestConsumer(t *testing.T) {

	Convey("Simple test with no NSQ", t, func() {
		c, err := NewConsumer("test", "test")

		if err != nil {
			fmt.Println("Failed to create a consumer", err)
			return
		}

		defer c.Stop()
		err = c.Start()
		if err != nil {
			fmt.Println("Failed to start server")
			return
		}
	})

	Convey("Simple test with NSQ", t, func() {
		conf := new(conf.NsqConf)
		conf.Host = "0.0.0.0"
		conf.Port = 4150
		NewCom(&conf)
		defer Close()

		c, err := NewConsumer("test", "test")

		if err != nil {
			fmt.Println("Failed to create a consumer", err)
			return
		}

		defer c.Stop()

		c.AddHandler(func(nm *nsq.Message) error { return nil })

		err = c.Start()
		if err != nil {
			fmt.Println("Failed to start consumer", err)
			return
		}

	})
}

func TestCommunicationAPI(t *testing.T) {

	// test channel
	output := make(chan []byte)

	conf := new(conf.NsqConf)
	conf.Host = "0.0.0.0"
	conf.Port = 4150
	NewCom(&conf)

	defer Close()

	// Producer
	NewProducer()

	// Consumer
	c, err := NewConsumer("test", "test")

	if err != nil {
		fmt.Println("Failed to create a consumer", err)
	}

	defer c.Stop()

	c.AddHandler(func(nm *nsq.Message) error {
		output <- nm.Body
		return nil
	})

	err = c.Start()
	if err != nil {
		fmt.Println("Failed to start consumer ", err)
		return
	}

	Convey("Sending a simple request", t, func() {
		msg := []byte("This is a simple test")

		Publish(Message{"test", msg})

		result := <-output

		fmt.Printf("Result %s\n", result)

		So(string(result), ShouldEqual, string(msg))
	})

}
