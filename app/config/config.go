/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */
package config

import (
	"github.com/intel/rsp-sw-toolkit-im-suite-utilities/configuration"
	"github.com/intel/rsp-sw-toolkit-im-suite-utilities/helper"
	log "github.com/sirupsen/logrus"
)

type (
	variables struct {
		ServiceName, LoggingLevel, Port                   string
		DbHost, DbPort, DbUser, DbPass, DbSSLMode, DbName string
		TelemetryEndpoint, TelemetryDataStoreName         string
		ResponseLimit                                     int
	}
)

// AppConfig exports all config variables
var AppConfig variables

// InitConfig loads application variables
func InitConfig() error {

	AppConfig = variables{}

	var err error

	config, err := configuration.NewConfiguration()
	errorHandler(err)

	AppConfig.ServiceName, err = config.GetString("serviceName")
	errorHandler(err)

	// size limit of RESTFul endpoints
	AppConfig.ResponseLimit, err = config.GetInt("responseLimit")
	errorHandler(err)

	AppConfig.DbHost, err = config.GetString("dbHost")
	errorHandler(err)

	AppConfig.DbPort, err = config.GetString("dbPort")
	errorHandler(err)

	AppConfig.DbUser, err = config.GetString("dbUser")
	errorHandler(err)

	AppConfig.DbName, err = config.GetString("dbName")
	errorHandler(err)

	AppConfig.DbSSLMode, err = config.GetString("dbSSLMode")
	errorHandler(err)

	AppConfig.DbPass, err = helper.GetSecret("dbPass")
	if err != nil {
		AppConfig.DbPass, err = config.GetString("dbPass")
		errorHandler(err)
	}

	// Webserver port
	AppConfig.Port, err = config.GetString("port")
	errorHandler(err)

	AppConfig.LoggingLevel, err = config.GetString("loggingLevel")
	errorHandler(err)

	AppConfig.TelemetryEndpoint, err = config.GetString("telemetryEndpoint")
	errorHandler(err)

	AppConfig.TelemetryDataStoreName, err = config.GetString("telemetryDataStoreName")
	errorHandler(err)

	return nil
}

func errorHandler(err error) {

	if err != nil {
		log.WithFields(log.Fields{
			"Method":  "config.InitConfig",
			"Action":  "Load config",
			"Message": "Unable to load config variables",
		}).Fatal(err.Error())
	}
}
