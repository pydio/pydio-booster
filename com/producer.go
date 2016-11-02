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

	"github.com/nsqio/go-nsq"
)

var (
	producer *nsq.Producer
)

// NewProducer that writes on the standard communication channel
func NewProducer() error {
	if !IsRunning() {
		return errors.New("NSQ must be running")
	}

	config := nsq.NewConfig()
	p, err := nsq.NewProducer(opts.TCPAddress, config)

	if err != nil {
		return err
	}

	producer = p
	return nil
}

// Publish a message to the standard communication channel
func Publish(m Message) error {
	err := producer.Publish(m.Topic, m.Content)

	return err
}

// StopProducer writing to the standard communication channel
func StopProducer() error {
	if !IsRunning() {
		return errors.New("NSQ must be running")
	}
	producer.Stop()

	return nil
}
