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
	"flag"
	"fmt"
	"log"
	"os"

	"runtime"

	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
	"github.com/mholt/caddy/caddytls"
	"github.com/pydio/pydio-booster/com"
	"github.com/pydio/pydio-booster/conf"
	"github.com/pydio/pydio-booster/scheduler"

	// List of plugins used in the soft
	_ "github.com/mholt/caddy/caddyhttp/basicauth"
	_ "github.com/mholt/caddy/caddyhttp/header"
	_ "github.com/mholt/caddy/caddyhttp/websocket"
	_ "github.com/pydio/pydio-booster/server/middleware/pydioadmin"
	_ "github.com/pydio/pydio-booster/server/middleware/pydioupload"
	_ "github.com/pydio/pydio-booster/server/middleware/pydiows"
)

// Flags that control program flow or startup
var (
	pydioconf string
	cpu       string
	logfile   string
	revoke    string
	version   bool
	plugins   bool
)

var directives = []string{
	"header",
	"basicauth",
	"pydioadmin",
	"pydioupload",
	"pydiows",
	"websocket",
}

func init() {
	caddy.TrapSignals()
	flag.BoolVar(&caddytls.Agreed, "agree", false, "Agree to Let's Encrypt Subscriber Agreement")
	flag.StringVar(&caddytls.DefaultCAUrl, "ca", "https://acme-v01.api.letsencrypt.org/directory", "Certificate authority ACME server")
	flag.StringVar(&pydioconf, "conf", "", "Configuration file to use (default="+caddy.DefaultConfigFile+")")
	flag.StringVar(&cpu, "cpu", "100%", "CPU cap")
	flag.BoolVar(&plugins, "plugins", false, "List installed plugins")
	flag.StringVar(&caddytls.DefaultEmail, "email", "", "Default Let's Encrypt account email address")
	flag.StringVar(&logfile, "log", "", "Process log file")
	flag.StringVar(&caddy.PidFile, "pidfile", "", "Path to write pid file")
	flag.BoolVar(&caddy.Quiet, "quiet", false, "Quiet mode (no initialization output)")
	flag.StringVar(&revoke, "revoke", "", "Hostname for which to revoke the certificate")
	flag.BoolVar(&version, "version", false, "Show version")

}

func main() {

	flag.Parse()

	// Run time definition
	runtime.GOMAXPROCS(runtime.NumCPU())

	caddy.AppName = "pydio"
	caddy.AppVersion = "local"

	// List all directives used and defined by pydio
	httpserver.RegisterDevDirective("pydioadmin", "")
	httpserver.RegisterDevDirective("pydioupload", "")
	httpserver.RegisterDevDirective("pydiows", "")

	if plugins {
		fmt.Println(caddy.DescribePlugins())
		os.Exit(0)
	}

	config, err := conf.LoadConfigurationFile(pydioconf)
	if err != nil {
		log.Fatal(err)
	}

	// Start your engines
	instance, err := caddy.Start(config.CaddyFile)
	if err != nil {
		log.Fatal(err)
	}

	if(config.Nsq != conf.NsqConf{}){
		// Starting the COM
		defer func() {
			com.Close()
			com.StopProducer()
		}()
		com.NewCom(&config.Nsq)
	}

	if(config.Scheduler != conf.SchedulerConf{}){
		scheduler.NewScheduler(&config.Scheduler)
	}

	// Twiddle your thumbs
	instance.Wait()
	com.StopProducer()
	log.Println("Exiting without listening!!")
}
