/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */
package productdata

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"testing"

	"github.com/intel/rsp-sw-toolkit-im-suite-product-data-service/app/config"
	"github.com/intel/rsp-sw-toolkit-im-suite-product-data-service/pkg/web"
)

func TestMain(m *testing.M) {

	if err := config.InitConfig(); err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())

}
func TestInsertSkuMapping(t *testing.T) {

	db := dbSetup(t)

	expectedMappings := insertSampleData(db, t)

	if expectedMappings == nil {
		t.Error("Unable to insert data")
	}
}

func TestInsertDuplicateProductIDsSKUMapping(t *testing.T) {

	db := dbSetup(t)
	expectedMappings := insertSampleDuplicateData(db, t)

	if expectedMappings == nil {
		t.Error("Unable to insert data")
	}

	testURL, err := url.Parse("http://localhost/test?$filter=sku eq 'DuplicateSku'")
	if err != nil {
		t.Error("Failed to parse test URL")
	}

	result, _, err := Retrieve(db, testURL.Query(), 100)
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

	db := dbSetup(t)

	insertSampleData(db, t)

	results, count, err := Retrieve(db, testURL.Query(), 1000)
	if err != nil {
		t.Error("Unable to retrieve SKUs")
	}
	if len(results) != 0 {
		t.Error("Expected results to be 0, but got something else")
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

	db := dbSetup(t)

	insertSampleData(db, t)

	results, count, err := Retrieve(db, testURL.Query(), 1000)
	if err != nil {
		t.Errorf("Unable to retrieve SKUs. Error: %s", err.Error())
	}

	if len(results) != 0 {
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

	db := dbSetup(t)

	insertSampleData(db, t)

	results, count, err := Retrieve(db, testURL.Query(), 1000)

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

	db := dbSetup(t)

	insertSampleData(db, t)

	_, _, err = Retrieve(db, testURL.Query(), 1000)
	if err != nil {
		t.Errorf("Retrieve failed with error %v", err.Error())
	}

}

func TestRetrieveSizeLimitWithTop(t *testing.T) {

	var sizeLimit = 1

	// Trying to return more than 1 result
	testURL, err := url.Parse("http://localhost/test?$inlinecount=allpages&$top=2")
	if err != nil {
		t.Error("Failed to parse test URL")
	}

	db := dbSetup(t)

	insertSampleData(db, t)

	results, count, err := Retrieve(db, testURL.Query(), sizeLimit)
	if err != nil {
		t.Errorf("Retrieve failed with error %v", err.Error())
	}

	if len(results) > sizeLimit {
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

	db := dbSetup(t)

	_, _, err = Retrieve(db, testURL.Query(), sizeLimit)
	if err == nil {
		t.Errorf("Expecting an error for invalid $top value")
	}

}

func TestRetrieveWithBadQuery(t *testing.T) {
	testURL, err := url.Parse("http://localhost/test?$filter=name eq ")
	if err != nil {
		t.Error("Failed to parse test URL")
	}

	db := dbSetup(t)

	insertSampleData(db, t)

	_, _, err = Retrieve(db, testURL.Query(), 1000)
	if err == nil {
		t.Error("Expected an error, but Retrieve function returned a value")
	}
}

func insertSampleData(db *sql.DB, t *testing.T) []SKUData {

	JSONSample := `[
		{ "sku":"MS122-32",
		  "productList": [ {"productId": "889319388921", "becomingReadable": 0.0456, "dailyTurn": 0.0121, "exitError": 0.0789, "metadata": {"color":"blue"} },
		  {"productId": "test", "becomingReadable": 0.0456, "dailyTurn": 0.0121, "exitError": 0.0789, "metadata": {"color":"blue"} }]
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

	if err := Insert(db, expectedMappings); err != nil {
		t.Error("Not able to insert into database: " + err.Error())
	}

	return expectedMappings
}

func insertSampleDuplicateData(db *sql.DB, t *testing.T) []SKUData {

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

	if err := Insert(db, expectedMappings); err != nil {
		t.Error("Not able to insert into database: " + err.Error())
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
	db := dbSetup(t)

	expectedMappings := make([]SKUData, 600)

	for i := 0; i < 600; i++ {
		mapObj := SKUData{SKU: "622738" + strconv.Itoa(i),
			ProductList: []ProductData{{ProductID: "123"}}}
		expectedMappings[i] = mapObj
	}

	if err := Insert(db, expectedMappings); err != nil {
		t.Error("Not able to insert into database: " + err.Error())
	}

}

func dbSetup(t *testing.T) *sql.DB {

	// Connect to PostgreSQL
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s", config.AppConfig.DbHost,
		config.AppConfig.DbPort,
		config.AppConfig.DbUser,
		config.AppConfig.DbName,
		config.AppConfig.DbSSLMode)
	if config.AppConfig.DbPass != "" {
		psqlInfo += " password=" + config.AppConfig.DbPass
	}

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		t.Fatal(err)
	}
	// Create table
	db.Exec(DbSchema)

	return db
}

func TestGetProductIDMetadataNotFound(t *testing.T) {
	db := dbSetup(t)

	result, err := GetProductMetadata(db, "00000000000000")
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
	db := dbSetup(t)

	InsertSampleProductMetadata(db, t)

	productID := "12345678912345"
	result, err := GetProductMetadata(db, productID)
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

func InsertSampleProductMetadata(db *sql.DB, t *testing.T) []SKUData {

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

	if err := Insert(db, expectedMappings); err != nil {
		t.Error("Not able to insert into database: " + err.Error())
	}

	return expectedMappings
}
