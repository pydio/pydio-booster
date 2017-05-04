// Package log contain the Pydio logger logic
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
package log

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

const (
	// Ldate in the local time zone: 2009/01/23
	Ldate = 1 << iota

	// Ltime in the local time zone: 01:23:23
	Ltime

	// Lmicroseconds resolution: 01:23:23.123123.  assumes Ltime.
	Lmicroseconds // microsecond resolution: 01:23:23.123123.  assumes Ltime.

	// Llongfile full file name and line number: /a/b/c/d.go:23
	Llongfile

	// Lshortfile final file name element and line number: d.go:23. overrides Llongfile
	Lshortfile

	// LUTC uses UTC rather than the local time zone, if Ldate or Ltime is set,
	LUTC

	// LstdFlags initial values for the standard logger
	LstdFlags = Ldate | Ltime

	// DEBUG Log Level
	DEBUG = 100

	// INFO Log Level
	INFO = 2

	// ERROR Log Level
	ERROR = 1
)

// Logger structure
type Logger struct {
	prefix string

	debugLogger *log.Logger
	infoLogger  *log.Logger
	errorLogger *log.Logger
}

var (
	logLevel      = INFO
	output        io.Writer
	defaultOutput io.Writer
	debugLogger   *log.Logger
	infoLogger    *log.Logger
	errorLogger   *log.Logger
)

func init() {
	output = os.Stdout
	defaultOutput = ioutil.Discard
	debugLogger = log.New(defaultOutput, "DEBUG ", log.Ldate|log.Ltime|log.Lmicroseconds)
	infoLogger = log.New(defaultOutput, "INFO ", log.Ldate|log.Ltime|log.Lmicroseconds)
	errorLogger = log.New(defaultOutput, "ERROR ", log.Ldate|log.Ltime|log.Lmicroseconds)
}

// New Pydio Logger
func New(logLevel int, prefix string, flag int) *Logger {

	debugOut, infoOut, errorOut := defaultOutput, defaultOutput, defaultOutput

	if logLevel >= DEBUG {
		debugOut = output
	}

	if logLevel >= INFO {
		infoOut = output
	}

	if logLevel >= ERROR {
		errorOut = output
	}

	logger := &Logger{
		prefix,

		log.New(debugOut, "DEBUG ", flag),
		log.New(infoOut, "INFO ", flag),
		log.New(errorOut, "ERROR ", flag),
	}

	return logger
}

// SetOutput default for the main logger
func SetOutput(w io.Writer) {
	output = w
}

// SetLevel default for the main logger
func SetLevel(l int) {

	logLevel = l

	debugOut, infoOut, errorOut := defaultOutput, defaultOutput, defaultOutput

	if logLevel >= DEBUG {
		debugOut = output
	}

	if logLevel >= INFO {
		infoOut = output
	}

	if logLevel >= ERROR {
		errorOut = output
	}

	log.Println("Set Level ", logLevel)

	debugLogger.SetOutput(debugOut)
	infoLogger.SetOutput(infoOut)
	errorLogger.SetOutput(errorOut)
}

// GetLevel of logging
func GetLevel() int {
	return logLevel
}

// Debugf replaces Printf
func Debugf(format string, v ...interface{}) {
	debugLogger.Printf("%s", fmt.Sprintf(format, v...))
}

// Debugln replaces Println
func Debugln(v ...interface{}) {
	debugLogger.Printf("%s", fmt.Sprintln(v...))
}

// Infof replaces Printf
func Infof(format string, v ...interface{}) {
	infoLogger.Printf("%s", fmt.Sprintf(format, v...))
}

// Infoln replaces Println
func Infoln(v ...interface{}) {
	infoLogger.Printf("%s", fmt.Sprintln(v...))
}

// Errorf replaces Printf
func Errorf(format string, v ...interface{}) {
	errorLogger.Printf("%s", fmt.Sprintf(format, v...))
}

// Errorln replaces Println
func Errorln(v ...interface{}) {
	errorLogger.Printf("%s", fmt.Sprintln(v...))
}

// Debugf replaces Printf
func (l *Logger) Debugf(format string, v ...interface{}) {
	l.debugLogger.Printf("%s%s", l.prefix, fmt.Sprintf(format, v...))
}

// Debugln replaces Println
func (l *Logger) Debugln(v ...interface{}) {
	l.debugLogger.Printf("%s", l.prefix+fmt.Sprintln(v...))
}

// Infof replaces Printf
func (l *Logger) Infof(format string, v ...interface{}) {
	l.infoLogger.Printf("%s%s", l.prefix, fmt.Sprintf(format, v...))
}

// Infoln replaces Println
func (l *Logger) Infoln(v ...interface{}) {
	l.infoLogger.Printf("%s", l.prefix+fmt.Sprintln(v...))
}

// Errorf replaces Printf
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.errorLogger.Printf("%s%s", l.prefix, fmt.Sprintf(format, v...))
}

// Errorln replaces Println
func (l *Logger) Errorln(v ...interface{}) {
	l.errorLogger.Printf("%s", l.prefix+fmt.Sprintln(v...))
}

// SetPrefix to the log messages
func (l *Logger) SetPrefix(p string) {
	l.prefix = p
}

// Output print function
func (l *Logger) Output(loglevel int, s string) error {
	if loglevel >= DEBUG {
		l.Debugln(s)
	} else if loglevel >= INFO {
		l.Infoln(s)
	} else if loglevel >= ERROR {
		l.Errorln(s)
	}

	return nil
}
