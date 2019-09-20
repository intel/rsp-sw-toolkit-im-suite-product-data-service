/*
 * INTEL CONFIDENTIAL
 * Copyright (2019) Intel Corporation.
 *
 * The source code contained or described herein and all documents related to the source code ("Material")
 * are owned by Intel Corporation or its suppliers or licensors. Title to the Material remains with
 * Intel Corporation or its suppliers and licensors. The Material may contain trade secrets and proprietary
 * and confidential information of Intel Corporation and its suppliers and licensors, and is protected by
 * worldwide copyright and trade secret laws and treaty provisions. No part of the Material may be used,
 * copied, reproduced, modified, published, uploaded, posted, transmitted, distributed, or disclosed in
 * any way without Intel/'s prior express written permission.
 * No license under any patent, copyright, trade secret or other intellectual property right is granted
 * to or conferred upon you by disclosure or delivery of the Materials, either expressly, by implication,
 * inducement, estoppel or otherwise. Any license under such intellectual property rights must be express
 * and approved by Intel in writing.
 * Unless otherwise agreed by Intel in writing, you may not remove or alter this notice or any other
 * notice embedded in Materials by Intel or Intel's suppliers or licensors in any way.
 */
package config

import (
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/configuration"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
)

type (
	variables struct {
		ServiceName, LoggingLevel, Port, ZeroMQ           string
		DbHost, DbPort, DbUser, DbPass, dbSSLmode, DbName string
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

	AppConfig.dbSSLmode, err = config.GetString("dbSSLmode")
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

	AppConfig.ZeroMQ, err = config.GetString("zeroMQ")
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
