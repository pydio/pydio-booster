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
	"context"
	"encoding/json"
	"io"
	"log"
	"net/url"

	"github.com/mholt/caddy/caddyhttp/httpserver"
	"github.com/pydio/pydio-booster/encoding/path"
	"github.com/pydio/pydio-booster/worker"
	"github.com/pydio/go/worker"
)

// NodeJob definition for the uploader
type NodeJob struct {
	HandleFunc func() error
}

// Do the job
func (j *NodeJob) Do() (err error) {
	return j.HandleFunc()
}

// NewNodeJob prepares the job for the middleware request
// based on the rules
func NewNodeJob(
	url url.URL,
	ctx context.Context,
	replacer httpserver.Replacer,
	encoder json.Encoder,
	writer io.Writer,
	close func() error,
	cancel func(),
) (pydioworker.Job, error) {

	job := &NodeJob{
		HandleFunc: func() error {
			defer close()

			// Retrieving the node
			q := &PathQuery{}

			if err := path.Unmarshal([]byte(url.Path), q); err != nil {
				log.Println("[ERROR:MW] NodeJob: ", err)
				cancel()
				return err
			}

			log.Printf("[INFO:MW] Node job: retrieved %v from %s", q.Node, url.Path)

			if err := encoder.Encode(q.Node); err != nil {
				log.Println("[ERROR:MW] NodeJob encoding : ", err)
				cancel()
				return err
			}

			return nil
		},
	}

	return job, nil
}
