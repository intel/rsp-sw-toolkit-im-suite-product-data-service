/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */
package main

import (
	"log"
	"os"
	"testing"

	"github.com/intel/rsp-sw-toolkit-im-suite-product-data-service/app/config"
)

//nolint :dupl
func TestMain(m *testing.M) {

	if err := config.InitConfig(); err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())

}

func TestDataProcess(t *testing.T) {

	db, err := dbSetup(config.AppConfig.DbHost,
		config.AppConfig.DbPort,
		config.AppConfig.DbUser, config.AppConfig.DbPass,
		config.AppConfig.DbName,
		config.AppConfig.DbSSLMode,
	)
	if err != nil {
		t.Fatal("Unable to connect to database")
	}

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

	if err := dataProcess(JSONSample, db); err != nil {
		t.Fatalf("error processing product data: %+v", err)
	}
}
