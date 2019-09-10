/**
 * INTEL CONFIDENTIAL
 * Copyright (2016, 2017) Intel Corporation.
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
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	golog "log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/jmoiron/sqlx"
	zmq "github.com/pebbe/zmq4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/product-data-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/product-data-service/app/productdata"
	"github.impcloud.net/RSP-Inventory-Suite/product-data-service/app/routes"
	"github.impcloud.net/RSP-Inventory-Suite/product-data-service/pkg/healthcheck"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
	reporter "github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics-influxdb"
)

const schema = `
CREATE TABLE IF NOT EXISTS skus (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	data JSONB	
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sku
ON skus ((data->'sku'));
`

func main() {

	// Ensure simple text format
	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	// Load config variables
	if err := config.InitConfig(); err != nil {
		log.WithFields(log.Fields{
			"Method": "config.InitConfig",
			"Action": "Load config",
		}).Fatal(err.Error())
	}

	// Initialize healthcheck
	healthCheck(config.AppConfig.Port)

	// Initialize metrics reporting
	initMetrics()

	setLoggingLevel(config.AppConfig.LoggingLevel)

	// Metrics
	mDbConnection := metrics.GetOrRegisterGauge(`Product-Data.Main.DB-Connection`, nil)
	mDbErr := metrics.GetOrRegisterGauge(`Product-Data.Main.DB-Error`, nil)

	log.WithFields(log.Fields{"Method": "main", "Action": "Start"}).Info("Starting application...")

	////////////////////////
	// Connect to PostgreSQL
	///////////////////////

	log.WithFields(log.Fields{"Method": "main", "Action": "Start"}).Info("Connecting to database...")

	db, err := dbSetup(config.AppConfig.ConnectionString, "5432", "postgres", "", config.AppConfig.DatabaseName)
	if err != nil {
		mDbErr.Update(1)
		log.WithFields(log.Fields{
			"Method":  "main",
			"Action":  "Start database",
			"Message": err.Error(),
		}).Fatal("Unable to connect to database.")
	}
	defer db.Close()
	mDbConnection.Update(1)

	// Receive data from EdgeX core data
	receiveZmqEvents(db)

	// Initiate webserver and routes
	startWebServer(db, config.AppConfig.Port, config.AppConfig.ResponseLimit, config.AppConfig.ServiceName)

	log.WithField("Method", "main").Info("Completed.")
}

func startWebServer(db *sql.DB, port string, responseLimit int, serviceName string) {

	// Start Webserver and pass additional data
	router := routes.NewRouter(db, responseLimit)

	// Create a new server and set timeout values.
	server := http.Server{
		Addr:           ":" + port,
		Handler:        router,
		ReadTimeout:    900 * time.Second,
		WriteTimeout:   900 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// We want to report the listener is closed.
	var wg sync.WaitGroup
	wg.Add(1)

	// Start the listener.
	go func() {
		log.Infof("%s running!", serviceName)
		log.Infof("Listener closed : %v", server.ListenAndServe())
		wg.Done()
	}()

	// Listen for an interrupt signal from the OS.
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt)

	// Wait for a signal to shutdown.
	<-osSignals

	// Create a context to attempt a graceful 5 second shutdown.
	const timeout = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Attempt the graceful shutdown by closing the listener and
	// completing all inflight requests.
	if err := server.Shutdown(ctx); err != nil {

		log.WithFields(log.Fields{
			"Method":  "main",
			"Action":  "shutdown",
			"Timeout": timeout,
			"Message": err.Error(),
		}).Error("Graceful shutdown did not complete")

		// Looks like we timedout on the graceful shutdown. Kill it hard.
		if err := server.Close(); err != nil {
			log.WithFields(log.Fields{
				"Method":  "main",
				"Action":  "shutdown",
				"Message": err.Error(),
			}).Error("Error killing server")
		}
	}

	// Wait for the listener to report it is closed.
	wg.Wait()
}

func healthCheck(port string) {

	isHealthyPtr := flag.Bool("isHealthy", false, "a bool, runs a healthcheck")
	flag.Parse()

	if *isHealthyPtr {
		status := healthcheck.Healthcheck(port)
		os.Exit(status)
	}

}

/*
dataProcess takes incoming product data and formats the data into
a JSON object that we can consume and use.

We are expecting the data to be passed to us in the following format:
"value": {
				"data": [
							{
								"sku": "12345679",
								"upc": "123456789783",
                "beingRead": 0.01,
                "becomingReadable": 0.02,
                "exitError": 0.03,
                "dailyTurn": 0.04
								"metadata": {
									"color":"blue",
									"size":"XS"
								}
							},
							{
								"sku": "12345679",
								"upc": "123456789784",
								"beingRead": "0.01",
								"becomingReadable": 0.02,
								"exitError": 0.03,
								"dailyTurn": 0.04
								"metadata": {
									"color":"red",
									"size":"M"
								}
							}
						],
				"sent_on": 1501872400247
  }

We will transform the data into the following format:

"sku" : "12345679",
"productList" : [
	{
		"productId" : "123456789783",
		"beingRead": 0.01,
		"becomingReadable": 0.02,
		"exitError": 0.03,
		"dailyTurn": 0.04
		"metadata" :
			{
				"color" : "blue",
				"size" : "XS"
			}
	},
	{
		"productId": "123456789784",
		"beingRead": "0.01",
		"becomingReadable": 0.02,
		"exitError": 0.03,
		"dailyTurn": 0.04
		"metadata":
			{
				"color":"red",
				"size":"M"
			}
	}
]

*/
func dataProcess(jsonBytes []byte, masterDB *sqlx.DB) error {
	// Metrics
	metrics.GetOrRegisterGauge(`Product-Data.dataProcess.Attempt`, nil).Update(1)
	mUnmarshalErr := metrics.GetOrRegisterGauge("Product-Data.dataProcess.Unmarshal-Error", nil)
	mMappingSkuCount := metrics.GetOrRegisterGaugeCollection("Product-Data.dataProcess.MappingData-SKU-Count", nil)
	mTotalLatency := metrics.GetOrRegisterTimer(`Product-Data.dataProcess.Total-Latency`, nil)
	/*
		TODO: Check with team on need to total latency.
		Per Instrumentation Guidance: Collect a timer for CPU intensive operations.
		[Should answer questions like:] How much time does your application spend rendering documents? Calculating hashes?
		(De)serializing JSON documents?
	*/

	startTime := time.Now()

	log.Debugf("Received data:\n%s", string(jsonBytes))

	var bv []productdata.IncomingData
	if err := json.Unmarshal(jsonBytes, &bv); err != nil {
		mUnmarshalErr.Update(1)
		return errors.Wrap(err, "unmarshal failed")
	}
	var incomingDataSlice = bv

	// Transform mapping.IncomingData to map of sku -> list of mapping.SKUData
	prodDataMap := make(map[string]productdata.SKUData)
	for _, item := range incomingDataSlice {
		productData := productdata.ProductData{
			ProductID:        item.ProductID,
			Metadata:         item.Metadata,
			BeingRead:        item.BeingRead,
			BecomingReadable: item.BecomingReadable,
			DailyTurn:        item.DailyTurn,
			ExitError:        item.ExitError,
		}
		skuData, repeatSKU := prodDataMap[item.SKU]
		if repeatSKU {
			skuData.ProductList = append(skuData.ProductList, productData)
		} else {
			skuData = productdata.SKUData{
				SKU:         item.SKU,
				ProductList: []productdata.ProductData{productData},
			}
			prodDataMap[item.SKU] = skuData
		}
	}

	// extract the values to a list
	prodDataList := make([]productdata.SKUData, 0, len(prodDataMap))
	for _, skuData := range prodDataMap {
		prodDataList = append(prodDataList, skuData)
	}

	// if err := productdata.Insert(masterDB, prodDataList); err != nil {
	// 	// Metrics not instrumented as it is handled in the controller.
	// 	return err
	// }

	log.WithFields(log.Fields{
		"Length": len(prodDataList),
		"Action": "Insert",
	}).Info("Product data inserted")

	mMappingSkuCount.Add(int64(len(prodDataList)))

	mTotalLatency.Update(time.Since(startTime))
	return nil
}

func initMetrics() {
	// setup metrics reporting
	if config.AppConfig.TelemetryEndpoint != "" {
		go reporter.InfluxDBWithTags(
			metrics.DefaultRegistry,
			time.Second*10,
			config.AppConfig.TelemetryEndpoint,
			config.AppConfig.TelemetryDataStoreName,
			"",
			"",
			nil,
		)
	}
}

func receiveZmqEvents(masterDB *sql.DB) {

	go func() {
		q, _ := zmq.NewSocket(zmq.SUB)
		defer q.Close()
		uri := fmt.Sprintf("%s://%s", "tcp", config.AppConfig.ZeroMQ)
		if err := q.Connect(uri); err != nil {
			logrus.Error(err)
		}
		logrus.Infof("Connected to 0MQ at %s", uri)
		// Edgex Delhi release uses no topic for all sensor data
		q.SetSubscribe("")

		for {
			msg, err := q.RecvMessage(0)
			if err != nil {
				id, _ := q.GetIdentity()
				logrus.Error(fmt.Sprintf("Error getting message %s", id))
				continue
			}
			for _, str := range msg {
				event := parseEvent(str)

				if event.Device != "SKU_Data_Device" {
					continue
				}
				for _, read := range event.Readings {

					if read.Name != "SKU_data" {
						continue
					}

					// data, err := base64.StdEncoding.DecodeString(read.Value)
					// if err != nil {
					// 	log.WithFields(log.Fields{
					// 		"Method": "receiveZmqEvents",
					// 		"Action": "product data ingestion",
					// 		"Error":  err.Error(),
					// 	}).Error("error decoding base64 value")
					// 	continue
					// }

					// if err := dataProcess(data, masterDB); err != nil {
					// 	log.WithFields(log.Fields{
					// 		"Method": "receiveZmqEvents",
					// 		"Action": "product data ingestion",
					// 		"Error":  err.Error(),
					// 	}).Error("error processing product data")
					// }

				}

			}

		}
	}()
}

func parseEvent(str string) *models.Event {
	event := models.Event{}

	if err := json.Unmarshal([]byte(str), &event); err != nil {
		logrus.Error(err.Error())
		logrus.Warn("Failed to parse event")
		return nil
	}
	return &event
}

func setLoggingLevel(loggingLevel string) {
	switch strings.ToLower(loggingLevel) {
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	// Not using filtered func (Info, etc ) so that message is always logged
	golog.Printf("Logging level set to %s\n", loggingLevel)
}

func dbSetup(host, port, user, password, dbname string) (*sql.DB, error) {

	// Connect to PostgreSQL
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Prepares database schema and indexes
	db.Exec(schema)

	return db, nil
}
