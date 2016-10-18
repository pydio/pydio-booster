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
package conf

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfLoader(t *testing.T) {

	Convey("Simple test for loading config file", t, func() {

		/*c, err := LoadConfigurationFile("../sample/conf_sample.json")

		if err != nil {
			fmt.Println("Failed to load config file", err)
			return
		}

		So(c.CaddyFilePath, ShouldEqual, "../sample/caddy_sample")
		So(c.CaddyFile, ShouldNotBeNil)

		So(c.Scheduler.Host, ShouldEqual, "localhost")
		So(c.Scheduler.TokenP, ShouldEqual, "token-public")
		So(c.Scheduler.TokenS, ShouldEqual, "token-secret")
		So(c.Scheduler.Minutes, ShouldEqual, 2)
		So(c.Nsq.Host, ShouldEqual, "0.0.0.0")
		So(c.Nsq.Port, ShouldEqual, 4150)*/

	})

}
