/*
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
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	db "github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
	"github.impcloud.net/RSP-Inventory-Suite/product-data-service/app/config"
)

var dbHost string

//nolint :dupl
func TestMain(m *testing.M) {

	if err := config.InitConfig(); err != nil {
		log.Fatal(err)
	}
	dbName := config.AppConfig.DatabaseName
	dbHost = config.AppConfig.ConnectionString + "/" + dbName

	os.Exit(m.Run())

}

func TestDataProcess(t *testing.T) {

	masterDb := createDB(t)
	defer masterDb.Close()

	JSONSample := []byte(`
				 [
							{
								"sku": "12345678",
								"upc": "123456789789",
								"beingRead": 0.01,
								"becomingReadable": 0.02,
								"exitError": 0.03,
								"dailyTurn": 0.04,
								"metadata": {
									"color":"blue",
									"size":"XS"
								}
							
							},
							{
								"sku": "12345679",
								"upc": "1234567891234",
								"metadata": {
									"color":"blue",
									"size":"XS"
								}
							},
							{
								"sku": "12345679",
								"upc": "123456789784",
								"dailyTurn": 0.04,
								"metadata": {
									"color":"red",
									"size":"M"
								}
							}
						]`)

	if err := dataProcess(JSONSample, masterDb); err != nil {
		t.Fatalf("error processing product data: %+v", err)
	}
}

func createDB(t *testing.T) *db.DB {
	masterDb, err := db.NewSession(dbHost, 1*time.Second)
	if err != nil {
		t.Fatalf("Unable to connect to db server: %+v", err)
	}

	return masterDb
}

func TestParseEvent(t *testing.T) {

	timestamp := 1471806386919

	eventStr := `{"origin":` + strconv.Itoa(timestamp) + `,
	"device":"rrs-gateway",
	"readings":[ {"name" : "gwevent", "value": " " } ] 
   }`

	event := parseEvent(eventStr)

	if event.Device != "rrs-gateway" || event.Origin != int64(timestamp) {
		t.Error("Error parsing edgex event")
	}

}
