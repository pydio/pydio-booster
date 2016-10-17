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
	"sync"

	"github.com/nsqio/go-nsq"
)

// Consumer of messages
type Consumer struct {
	URL       string
	Topic     string
	Channel   string
	Handlers  []func(message *nsq.Message) error
	Connected bool
	queue     *nsq.Consumer
	// Config ?
}

var (
	consumerWg sync.WaitGroup
)

// NewConsumer that will register a handler for different topic and channels
func NewConsumer(topic string, channel string) (*Consumer, error) {

	connected := false

	config := nsq.NewConfig()
	q, err := nsq.NewConsumer(topic, channel, config)
	if err != nil {
		return nil, err
	}

	opts := getOptions()

	c := &Consumer{
		URL:       opts.TCPAddress,
		Topic:     topic,
		Channel:   channel,
		Connected: connected,
		queue:     q,
	}

	return c, nil
}

// WaitForConsumers to be closed before continuing
func WaitForConsumers() {
	consumerWg.Wait()
}

// Start the consumer main handler
func (c *Consumer) Start() error {

	err := c.queue.ConnectToNSQD(c.URL)
	if err != nil {
		return err
	}

	// Making sure the consumer stops if nsq if everything stops
	go func() {
		defer c.queue.Stop()

		consumerWg.Wait()
	}()

	c.Connected = true

	consumerWg.Add(1)

	return nil
}

// AddHandler to the message consumer
func (c *Consumer) AddHandler(handler func(*nsq.Message) error) {
	c.queue.AddHandler(nsq.HandlerFunc(handler))

	c.Handlers = append(c.Handlers, handler)
}

//Stop consumer listening
func (c *Consumer) Stop() {
	if !c.Connected {
		return
	}

	c.queue.Stop()

	consumerWg.Done()
}
