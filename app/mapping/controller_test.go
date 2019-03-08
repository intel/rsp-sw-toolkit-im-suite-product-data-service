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
package mapping

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	db "github.impcloud.net/Responsive-Retail-Core/mongodb"
	"github.impcloud.net/RSP-Inventory-Suite/product-data-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/product-data-service/pkg/web"
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

func TestInsertSkuMapping(t *testing.T) {
	dbName := config.AppConfig.DatabaseName
	dbHost = config.AppConfig.ConnectionString + "/" + dbName

	masterDb, err := db.NewSession(dbHost, 5*time.Second)
	if err != nil {
		t.Error("Unable to connect to db")
	}
	defer masterDb.Close()

	expectedMappings := insertSampleData(masterDb, t)

	if expectedMappings == nil {
		t.Error("Unable to insert data")
	}
}

func TestInsertDuplicateProductIDsSKUMapping(t *testing.T) {

	dbName := config.AppConfig.DatabaseName
	dbHost = config.AppConfig.ConnectionString + "/" + dbName

	masterDb, err := db.NewSession(dbHost, 5*time.Second)
	if err != nil {
		t.Error("Unable to connect to db")
	}
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	expectedMappings := insertSampleDuplicateData(masterDb, t)

	if expectedMappings == nil {
		t.Error("Unable to insert data")
	}

	testURL, err := url.Parse("http://localhost/test?$filter=sku eq 'DuplicateSku'")
	if err != nil {
		t.Error("Failed to parse test URL")
	}

	result, _, err := Retrieve(copySession, testURL.Query(), 100)
	if err != nil {
		t.Fatalf("Error reteiving SKUs: %s", err.Error())
	}

	bytes, _ := json.Marshal(result)
	var skus []SKUData
	if err := json.Unmarshal(bytes, &skus); err != nil {
		t.Fatal(err)
	}

	if len(skus) != 1 {
		t.Error("Duplicate SKU not found after insert, or found too many SKUs")
	}

	if len(skus[0].ProductList) != 1 {
		t.Error("Duplicate ProductIDs not removed")
	}
}

func TestRetrieveCount(t *testing.T) {
	testURL, err := url.Parse("http://localhost/test?$count")
	if err != nil {
		t.Error("Failed to parse test URL")
	}

	masterDb := createDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	insertSampleData(masterDb, t)

	results, count, err := Retrieve(copySession, testURL.Query(), 1000)
	if err != nil {
		t.Error("Unable to retrieve SKUs")
	}
	if results != nil {
		t.Error("Expected results to be nil, but got something else")
	}

	if count == nil {
		t.Error("Expected count type, but got something else")
	}
}

func TestRetrieveCountWithFilterQuery(t *testing.T) {
	testURL, err := url.Parse("http://localhost/test?$count&$filter=sku eq 'MS122-32'")
	if err != nil {
		t.Error("Failed to parse test URL")
	}

	masterDb := createDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	insertSampleData(masterDb, t)

	results, count, err := Retrieve(copySession, testURL.Query(), 1000)
	if err != nil {
		t.Error("Unable to retrieve SKUs")
	}

	if results != nil {
		t.Error("Expected results to be nil, but got something else")
	}

	if count == nil {
		t.Error("Expected count type, but got something else")
	}
}

func TestRetrieveInlinecount(t *testing.T) {

	testURL, err := url.Parse("http://localhost/test?$filter=sku eq 'MS122-32'&$inlinecount=allpages")
	if err != nil {
		t.Error("failed to parse test url")
	}

	masterDb := createDB(t)
	defer masterDb.Close()

	insertSampleData(masterDb, t)

	copySession := masterDb.CopySession()
	defer copySession.Close()

	results, count, err := Retrieve(copySession, testURL.Query(), 1000)

	if count == nil {
		t.Error("expecting inlinecount result")
	}

	if results == nil {
		t.Error("Expecting results, but got nil")
	}

	if err != nil {
		t.Error("Unable to retrieve", err.Error())
	}
}

func TestRetrieveCountNoDatabase(t *testing.T) {
	testURL, err := url.Parse("http://localhost/test?$count")
	if err != nil {
		t.Error("Failed to parse test URL")
	}

	// No database session, so query should fail
	if _, _, err = Retrieve(nil, testURL.Query(), 1000); err == nil {
		t.Error("Expected an error, but Retrieve returned a value")
	}
}

func TestRetrieveWithFilter(t *testing.T) {
	testURL, err := url.Parse("http://localhost/test?$filter=name eq 'asdf'")
	if err != nil {
		t.Error("Failed to parse test URL")
	}

	masterDb := createDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	insertSampleData(masterDb, t)

	result, _, err := Retrieve(copySession, testURL.Query(), 1000)
	if err != nil {
		t.Errorf("Retrieve failed with error %v", err.Error())
	}

	if _, ok := result.([]interface{}); !ok {
		t.Errorf("Expected []interface{}, but got %T", result)
	}
}

func TestRetrieveSizeLimitWithTop(t *testing.T) {

	var sizeLimit = 1

	// Trying to return more than 1 result
	testURL, err := url.Parse("http://localhost/test?$inlinecount=allpages&$top=2")
	if err != nil {
		t.Error("Failed to parse test URL")
	}

	masterDb := createDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	insertSampleData(masterDb, t)

	results, count, err := Retrieve(copySession, testURL.Query(), sizeLimit)
	if err != nil {
		t.Errorf("Retrieve failed with error %v", err.Error())
	}

	resultSlice := reflect.ValueOf(results)

	if resultSlice.Len() > sizeLimit {
		t.Errorf("Error retrieving results with size limit. Expected: %d , received: %d", sizeLimit, count.Count)
	}

}

func TestRetrieveSizeLimitInvalidTop(t *testing.T) {

	var sizeLimit = 1

	// Trying to return more than 1 result
	testURL, err := url.Parse("http://localhost/test?$inlinecount=allpages&$top=string")
	if err != nil {
		t.Error("Failed to parse test URL")
	}

	masterDb := createDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	_, _, err = Retrieve(copySession, testURL.Query(), sizeLimit)
	if err == nil {
		t.Errorf("Expecting an error for invalid $top value")
	}

}

func TestRetrieveWithBadQuery(t *testing.T) {
	testURL, err := url.Parse("http://localhost/test?$filter=name eq ")
	if err != nil {
		t.Error("Failed to parse test URL")
	}

	masterDb := createDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	insertSampleData(masterDb, t)

	_, _, err = Retrieve(copySession, testURL.Query(), 1000)
	if err == nil {
		t.Error("Expected an error, but Retrieve returned a value")
	}
}

func TestRetrieveWithNoDb(t *testing.T) {
	testURL, err := url.Parse("http://localhost/test?$filter=name eq 'asdf'")
	if err != nil {
		t.Error("Failed to parse test URL")
	}

	_, _, err = Retrieve(nil, testURL.Query(), 1000)
	if err == nil {
		t.Error("Expected an error, but Retrieve returned a value")
	}
}

func insertSampleData(db *db.DB, t *testing.T) []SKUData {

	copySession := db.CopySession()
	defer copySession.Close()

	JSONSample := `[
		{ "sku":"MS122-32",
		  "productList": [ {"productId": "889319388921", "becomingReadable": 0.0456, "dailyTurn": 0.0121, "exitError": 0.0789, "metadata": {"color":"blue"} } ]
		},
		{ "sku":"MS122-33",
			"productList": [ {"productId": "889319388922", "becomingReadable": 0.0456, "beingRead": 0.0123, "dailyTurn": 0.0121, "exitError": 0.0789, "metadata": {"color":"blue"} } ]
		},
		{ "sku":"MS122-34",
			"productList": [ {"productId": "889319388923", "becomingReadable": 0.0456, "beingRead": 0.0123, "dailyTurn": 0.0121, "exitError": 0.0789, "metadata": {"color":"blue"} } ]
		}
	]`

	var expectedMappings []SKUData
	err := json.Unmarshal([]byte(JSONSample), &expectedMappings)
	if err != nil {
		t.Fatal("Not able to Unmarshal JSON object: " + err.Error())
	}

	if err := Insert(copySession, expectedMappings); err != nil {
		t.Error("Not able to insert into mongodb: " + err.Error())
	}

	return expectedMappings
}

func insertSampleDuplicateData(db *db.DB, t *testing.T) []SKUData {

	copySession := db.CopySession()
	defer copySession.Close()

	JSONSample := `[
		{ "sku":"DuplicateSku",
		  "productList": [ 
				{"productId": "889319388921", "metadata": {"color":"blue"} },
				{"productId": "889319388921", "metadata": {"color":"blue"} },
				{"productId": "889319388921", "metadata": {"color":"blue"} }
			]
		}	
	]`

	var expectedMappings []SKUData
	err := json.Unmarshal([]byte(JSONSample), &expectedMappings)
	if err != nil {
		t.Fatal("Not able to Unmarshal JSON object: " + err.Error())
	}

	if err := Insert(copySession, expectedMappings); err != nil {
		t.Error("Not able to insert into mongodb: " + err.Error())
	}

	return expectedMappings
}

func TestRemoveDuplicateProductID(t *testing.T) {

	duplicateIDs := []ProductData{{ProductID: "889319388921"},
		{ProductID: "889319388921"},
		{ProductID: "889319388921"}}
	expectedIDs := []ProductData{{ProductID: "889319388921"}}

	newList := removeDuplicateProducts(duplicateIDs)

	if len(newList) != len(expectedIDs) {
		t.Error("Not able to remove duplicates correctly")
	}
}

func TestBulkInsert(t *testing.T) {
	masterDb := createDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	expectedMappings := make([]SKUData, 600)

	for i := 0; i < 600; i++ {
		mapObj := SKUData{SKU: "622738" + strconv.Itoa(i),
			ProductList: []ProductData{{ProductID: "123"}}}
		expectedMappings[i] = mapObj
	}

	if err := Insert(copySession, expectedMappings); err != nil {
		t.Error("Not able to insert into mongodb: " + err.Error())
	}

}

func createDB(t *testing.T) *db.DB {
	masterDb, err := db.NewSession(dbHost, 10*time.Second)
	if err != nil {
		t.Fatal("Unable to connect to db server")
	}

	return masterDb
}

func TestGetProductIDMetadataNotFound(t *testing.T) {
	masterDb := createDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	result, err := GetProductMetadata(copySession, "00000000000000")
	if err != nil {
		if common, ok := err.(web.CommonError); !ok || common.Code != http.StatusNotFound {
			t.Errorf("Retrieve failed with error %+v", err)
		}
	}

	if result.ProductList != nil {
		t.Errorf("Expected result.ProductList to be nil, but got %v", result)
	}
}

func TestGetProductIDMetadataFound(t *testing.T) {
	masterDb := createDB(t)
	defer masterDb.Close()

	copySession := masterDb.CopySession()
	defer copySession.Close()

	InsertSampleProductMetadata(masterDb, t)

	productID := "12345678912345"
	result, err := GetProductMetadata(copySession, productID)
	if err != nil {
		t.Errorf("Retrieve failed with error %+v", err)
	}

	if result.ProductList == nil {
		t.Fatal("Expected to find a ProductList item, but ProductList nil")
	}

	if len(result.ProductList) > 1 {
		t.Fatal("ProductList should only contain one element")
	}

	if result.ProductList[0].ProductID != productID {
		t.Errorf("ProductID did not match expected ProductID: %s received: %s",
			productID, result.ProductList[0].ProductID)
	}
}

func InsertSampleProductMetadata(db *db.DB, t *testing.T) []SKUData {

	copySession := db.CopySession()
	defer copySession.Close()

	JSONSample := `[
		{ "sku":"MS122-33", "name":"mens formal pants",  
		  "productList": [ {"productId": "12345678912345", "metadata": {"color":"blue"} } ]
		},
		{ "sku":"MS122-34", "name":"mens formal pants",  
			"productList": [ {"productId": "12345678912346", "metadata": {"color":"blue"} } ]
		}
	]`

	var expectedMappings []SKUData
	err := json.Unmarshal([]byte(JSONSample), &expectedMappings)
	if err != nil {
		t.Fatal("Not able to Unmarshal JSON object: " + err.Error())
	}

	if err := Insert(copySession, expectedMappings); err != nil {
		t.Error("Not able to insert into mongodb: " + err.Error())
	}

	return expectedMappings
}
