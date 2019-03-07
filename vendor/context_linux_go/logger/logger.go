//
// INTEL CONFIDENTIAL
// Copyright 2017 Intel Corporation.
//
// This software and the related documents are Intel copyrighted materials, and your use of them is governed
// by the express license under which they were provided to you (License). Unless the License provides otherwise,
// you may not use, modify, copy, publish, distribute, disclose or transmit this software or the related documents
// without Intel's prior written permission.
//
// This software and the related documents are provided as is, with no express or implied warranties, other than
// those that are expressly stated in the License.
//

package logger

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

// Params is a map to create parameters to log
type Params logrus.Fields

// LogLevel defines severity of logged entries
type LogLevel uint32

const (
	// PanicLevel level, highest level of severity. Logs and then calls panic with the
	// message passed to Debug, Info, ...
	PanicLevel LogLevel = iota
	// FatalLevel level. Logs and then calls `os.Exit(1)`. It will exit even if the
	// logging level is set to Panic.
	FatalLevel
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	// Commonly used for hooks to send errors to an error tracking service.
	ErrorLevel
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel
	// InfoLevel level. General operational entries about what's going on inside the
	// application.
	InfoLevel
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	DebugLevel
)

// ContextGoLogger structure holds logrus instance, tag and writer
type ContextGoLogger struct {
	logger *logrus.Logger
	tag    string
	writer io.Writer
}

// LogFormat defines which format should be used for log entries
type LogFormat uint32

const (
	// TextFormat is a log format that formats the outputs as plain text
	TextFormat LogFormat = iota
	// JSONFormat is a log format that formats the outputs as JSON
	JSONFormat
)

// GlobalLogger instance that can be used by developer
var GlobalLogger = New("global", TextFormat, os.Stdout, DebugLevel)

// New function creates new instance with formatter
// tag: string type to add to each log message for the instance
// log formatter: Currently it supports TextFormat, JSONFormat
// Output : output typ Ex: file, os.Stdout etc
// logLevel : Minimum level of logs to save
func New(tag string, logFormatter LogFormat, output io.Writer, logLevel LogLevel) *ContextGoLogger {

	var logging = &ContextGoLogger{
		logger: logrus.New(),
		tag:    tag,
	}

	logging.SetLogFormatter(logFormatter)
	logging.SetOutput(output)
	logging.SetLogLevel(logLevel)

	return logging
}

// SetLogFormatter is a log format setter
func (loggerInstance *ContextGoLogger) SetLogFormatter(logFormatter LogFormat) {
	switch logFormatter {
	case TextFormat:
		loggerInstance.logger.Formatter = new(logrus.TextFormatter)
	case JSONFormat:
		loggerInstance.logger.Formatter = new(logrus.JSONFormatter)
	}
}

//SetOutput is a output type setter
func (loggerInstance *ContextGoLogger) SetOutput(output io.Writer) {
	loggerInstance.logger.Out = output
	loggerInstance.writer = output
}

//DisableLogging allows to disable logging
func (loggerInstance *ContextGoLogger) DisableLogging() {
	loggerInstance.logger.Out = ioutil.Discard
}

//EnableLogging allows to enable logging after disabling it
func (loggerInstance *ContextGoLogger) EnableLogging() {
	loggerInstance.logger.Out = loggerInstance.writer
}

//SetLogLevel allows to change log level
func (loggerInstance *ContextGoLogger) SetLogLevel(logLevel LogLevel) {

	switch logLevel {
	case DebugLevel:
		loggerInstance.logger.Level = logrus.DebugLevel
	case InfoLevel:
		loggerInstance.logger.Level = logrus.InfoLevel
	case WarnLevel:
		loggerInstance.logger.Level = logrus.WarnLevel
	case ErrorLevel:
		loggerInstance.logger.Level = logrus.ErrorLevel
	case FatalLevel:
		loggerInstance.logger.Level = logrus.FatalLevel
	case PanicLevel:
		loggerInstance.logger.Level = logrus.PanicLevel
	}
}

//SetLogFile allows to set logger to log to file specified as parameter
func (loggerInstance *ContextGoLogger) SetLogFile(logFile string) {
	loggerInstance.logToFile(logFile)
}

//private method to convert input to logrus fields
func (loggerInstance *ContextGoLogger) getData(params Params) logrus.Fields {
	data := make(logrus.Fields)
	for k, v := range params {
		data[k] = v
	}
	data["TAG"] = loggerInstance.tag
	return data
}

//Debug level. Usually only enabled when debugging. Very verbose logging.
//message : String
//params: Params of Type maps that user can log any number of key value pairs
func (loggerInstance *ContextGoLogger) Debug(message string, params Params) {
	loggerInstance.logger.WithFields(loggerInstance.getData(params)).Debug(message)
}

//Info level. General operational entries about what's going on inside the application.
//message : String
//params: Params of Type maps that user can log any number of key value pairs
func (loggerInstance *ContextGoLogger) Info(message string, params Params) {
	loggerInstance.logger.WithFields(loggerInstance.getData(params)).Info(message)
}

//Warn level. Non-critical entries that deserve eyes.
//message : String
//params: Params of Type maps that user can log any number of key value pairs
func (loggerInstance *ContextGoLogger) Warn(message string, params Params) {
	loggerInstance.logger.WithFields(loggerInstance.getData(params)).Warn(message)
}

//Error level. Logs. Used for errors that should definitely be noted.
//message : String
//params: Params of Type maps that user can log any number of key value pairs
func (loggerInstance *ContextGoLogger) Error(message string, params Params) {
	loggerInstance.logger.WithFields(loggerInstance.getData(params)).Error(message)
}

//Fatal level. Logs and then calls `os.Exit(1)`. It will exit even if the logging
//level is set to Panic.
//message : String
//params: Params of Type maps that user can log any number of key value pairs
func (loggerInstance *ContextGoLogger) Fatal(message string, params Params) {
	loggerInstance.logger.WithFields(loggerInstance.getData(params)).Fatal(message)
}

//Panic level, highest level of severity.
//message : String
//params: Params of Type maps that user can log any number of key value pairs
func (loggerInstance *ContextGoLogger) Panic(message string, params Params) {
	loggerInstance.logger.WithFields(loggerInstance.getData(params)).Panic(message)
}

//Log to file will take input path to create/append logs to existing file
func (loggerInstance *ContextGoLogger) logToFile(pathToFile string) {
	file, err := os.OpenFile(pathToFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err == nil {
		loggerInstance.SetOutput(file)
	} else {
		loggerInstance.Info("Failed to log to file, using default stdout", nil)
	}
}

// StackTrace logs the current stack trace at Error level
func (loggerInstance *ContextGoLogger) StackTrace(message string, err interface{}) {
	loggerInstance.logger.Errorln(fmt.Sprintf("%s: %v", message, err))
	pc := make([]uintptr, 10)
	n := runtime.Callers(0, pc)
	if n == 0 {
		return
	}
	pc = pc[:n]
	frames := runtime.CallersFrames(pc)
	for {
		frame, more := frames.Next()
		if strings.Contains(frame.File, "runtime/") {
			continue
		}
		loggerInstance.logger.Errorln(fmt.Sprintf("%v - %v:%v", frame.Function, frame.File, frame.Line))
		if !more {
			break
		}
	}
}
