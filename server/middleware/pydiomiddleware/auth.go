// Package pydiomiddleware contains the logic for a middleware directive (repetitive task done for a Pydio request)
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
	"net/url"

	pydhttp "github.com/pydio/pydio-booster/http"
	pydioworker "github.com/pydio/pydio-booster/worker"
)

// AuthJob definition for the uploader
type AuthJob struct {
	HandleFunc func() error
}

// Do the job
func (j *AuthJob) Do() (err error) {
	return j.HandleFunc()
}

// NewAuthJob prepares the job for the middleware request
// based on the rules
func NewAuthJob(
	ctx context.Context,
	url url.URL,
	encoder Encoder,
	close func() error,
	cancel func(),
) (pydioworker.Job, error) {

	job := &AuthJob{
		HandleFunc: func() error {
			defer close()

			query := url.Query()

			a := &pydhttp.Auth{
				Token: query.Get("auth_token"),
				Hash:  query.Get("auth_hash"),
			}

			err := encoder.Encode(a)
			if err != nil {
				logger.Errorln("Could not encode auth")
				cancel()
			}

			logger.Debugln("User token ", a)

			return nil
		},
	}

	return job, nil
}
