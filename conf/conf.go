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
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
)

const (
	// VersionString gives the latest verson details
	VersionString string = "##BUILD_VERSION_STRING##"

	// VersionDate gives the latest verson date of release
	VersionDate string = "##BUILD_VERSION_DATE##"
)

// NsqConf definition
type NsqConf struct {
	Host string
	Port int
}

// SchedulerConf definition
type SchedulerConf struct {
	Host    string
	TokenP  string
	TokenS  string
	Minutes int
}

// Configuration object
type Configuration struct {
	CaddyFilePath string
	Scheduler     SchedulerConf
	Nsq           NsqConf
	CaddyFile     caddy.Input
}

// LoadConfigurationFile into the caddy main file
func LoadConfigurationFile(confFilePath string) (*Configuration, error) {

	file, _ := os.Open(confFilePath)
	decoder := json.NewDecoder(file)
	configuration := &Configuration{}
	err := decoder.Decode(configuration)
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(configuration.CaddyFilePath, "./") {
		// Simply look for caddy file in the same folder as main config file
		confFilePathDir := filepath.Dir(confFilePath)
		caddyFileName := filepath.Base(configuration.CaddyFilePath)
		configuration.CaddyFilePath = filepath.Join(confFilePathDir, caddyFileName)
	}
	err = loadCaddyfile(configuration)
	if err != nil {
		return nil, err
	}
	return configuration, nil
}

func loadCaddyfile(configuration *Configuration) error {
	conf := configuration.CaddyFilePath
	// Try -conf flag
	if conf != "" {
		if conf == "stdin" {
			configuration.CaddyFile, _ = caddy.CaddyfileFromPipe(os.Stdin, "http")
			return nil
		}

		contents, err := ioutil.ReadFile(conf)
		if err != nil {
			return err
		}

		configuration.CaddyFile = caddy.CaddyfileInput{
			Contents:       contents,
			Filepath:       conf,
			ServerTypeName: "http",
		}
		return nil
	}

	// command line args
	if flag.NArg() > 0 {
		confBody := httpserver.Host + ":" + httpserver.Port + "\n" + strings.Join(flag.Args(), "\n")
		configuration.CaddyFile = caddy.CaddyfileInput{
			Contents: []byte(confBody),
			Filepath: "args",
		}
		return nil
	}

	// Caddyfile in cwd
	contents, err := ioutil.ReadFile(caddy.DefaultConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			configuration.CaddyFile = caddy.DefaultInput("http")
			return nil
		}
		return err
	}
	configuration.CaddyFile = caddy.CaddyfileInput{
		Contents:       contents,
		Filepath:       caddy.DefaultConfigFile,
		ServerTypeName: "http",
	}
	return nil
}
