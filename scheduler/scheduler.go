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
package scheduler

import (
	"net/http"

	"io"
	"log"
	"net/url"
	"os"

	"github.com/jasonlvhit/gocron"
	pydconf "github.com/pydio/pydio-booster/conf"
	pydhttp "github.com/pydio/pydio-booster/http"
)

var (
	schedulerConf *pydconf.SchedulerConf
)

func pydioMasterScheduler() {

	log.Println("Triggering pydio scheduler master command")

	host := schedulerConf.Host
	tokenP := schedulerConf.TokenP
	tokenS := schedulerConf.TokenS

	url, err := url.Parse(host + "/api/ajxp_conf/scheduler_runAll")
	if err != nil {
		log.Printf("Error parsing url, exiting task")
		return
	}

	token := pydhttp.NewToken(tokenP, tokenS)
	// Building Query
	args := token.GetQueryArgs(url.Path)
	values := url.Query()
	values.Add("auth_hash", args.Hash)
	values.Add("auth_token", args.Token)
	url.RawQuery = values.Encode()

	//log.Printf("URL is -%s- -%s- ", url.Path, url.String())

	request, _ := http.NewRequest("GET", url.String(), nil)
	//log.Println("Sending request ", request)

	client := pydhttp.NewClient()
	response, err := client.Do(request)

	if err != nil {
		log.Printf("Error while trying to execute request")
	} else {
		defer response.Body.Close()
		_, err := io.Copy(os.Stdout, response.Body)
		if err != nil {
			log.Printf("Error while reading request response body")
		}
	}

}

func NewScheduler(conf *pydconf.SchedulerConf) error {

	schedulerConf = conf

	gocron.Every(uint64(conf.Minutes)).Minutes().Do(pydioMasterScheduler)

	go func() {
		<-gocron.Start()
	}()

	return nil
}
