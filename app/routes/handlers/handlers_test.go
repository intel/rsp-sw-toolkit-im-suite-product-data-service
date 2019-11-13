// /*
//  * INTEL CONFIDENTIAL
//  * Copyright (2019) Intel Corporation.
//  *
//  * The source code contained or described herein and all documents related to the source code ("Material")
//  * are owned by Intel Corporation or its suppliers or licensors. Title to the Material remains with
//  * Intel Corporation or its suppliers and licensors. The Material may contain trade secrets and proprietary
//  * and confidential information of Intel Corporation and its suppliers and licensors, and is protected by
//  * worldwide copyright and trade secret laws and treaty provisions. No part of the Material may be used,
//  * copied, reproduced, modified, published, uploaded, posted, transmitted, distributed, or disclosed in
//  * any way without Intel/'s prior express written permission.
//  * No license under any patent, copyright, trade secret or other intellectual property right is granted
//  * to or conferred upon you by disclosure or delivery of the Materials, either expressly, by implication,
//  * inducement, estoppel or otherwise. Any license under such intellectual property rights must be express
//  * and approved by Intel in writing.
//  * Unless otherwise agreed by Intel in writing, you may not remove or alter this notice or any other
//  * notice embedded in Materials by Intel or Intel's suppliers or licensors in any way.
//  */
package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/product-data-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/product-data-service/app/productdata"
	"github.impcloud.net/RSP-Inventory-Suite/product-data-service/pkg/web"
)

type inputTest struct {
	input []byte
	code  int
}

func TestMain(m *testing.M) {

	if err := config.InitConfig(); err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())

}

func TestGetSkuMapping(t *testing.T) {

	testCases := []string{
		"/skus?$filter=name eq 'mens golf pants'",
		"/skus",
		"/skus?$filter=startswith(sku,'m')&$inlinecount=allpages",
		"/skus?$filter=startswith(sku,'m')&$count",
	}

	db := dbSetup(t)

	for _, item := range testCases {

		request, err := http.NewRequest("GET", item, nil)
		if err != nil {
			t.Errorf("Unable to create new HTTP request %s", err.Error())
		}

		recorder := httptest.NewRecorder()

		mapp := Mapping{db, 1000}

		handler := web.Handler(mapp.GetSkuMapping)

		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK &&
			recorder.Code != http.StatusNoContent {
			t.Errorf("Success expected: %d Actual: %d", http.StatusOK, recorder.Code)
		}
	}
}

func TestGetSkuMappingNegative(t *testing.T) {

	testCases := []string{
		"/skus?$filter=startswith(sku,'m')&$count&$inlinecount=allpages",
	}

	db := dbSetup(t)

	for _, item := range testCases {

		request, err := http.NewRequest("GET", item, nil)
		if err != nil {
			t.Errorf("Unable to create new HTTP request %s", err.Error())
		}

		recorder := httptest.NewRecorder()

		mapp := Mapping{db, 1000}

		handler := web.Handler(mapp.GetSkuMapping)

		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Error expected: %d Actual: %d", http.StatusBadRequest, recorder.Code)
		}
	}
}

func TestGetIndex(t *testing.T) {
	request, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Errorf("Unable to create new HTTP request %s", err.Error())
	}
	recorder := httptest.NewRecorder()
	mapp := Mapping{nil, 1000}
	handler := web.Handler(mapp.Index)
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected 200 response")
	}
	log.Print(recorder.Body.String())
	if recorder.Body.String() != "\"Product Data Service\"" {
		t.Errorf("Expected body to equal Product Data Service")
	}
}

func testHandlerHelper(input []inputTest, handler web.Handler, t *testing.T) {
	t.Helper()

	for _, item := range input {
		request, err := http.NewRequest("POST", "/skus",
			bytes.NewBuffer(item.input))
		if err != nil {
			t.Errorf("Unable to create new HTTP Request: %+v", err)
		}

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		if recorder.Code != item.code {
			t.Errorf("Status code didn't match\n"+
				"data: %s\n"+
				"status code expected: %d, received: %d\n"+
				"body: %s",
				item.input, item.code, recorder.Code, recorder.Body.String())
		}
	}
}

func TestInsertMapping(t *testing.T) {
	db := dbSetup(t)

	var JSONSample = []inputTest{
		{
			input: []byte(`{
						"data":
								[{	"sku":"MS122-34",
									"productList":[
										{"productId": "886602023377", "metadata": {"color":"blue"} },
										{"productId": "888446970466", "metadata": {"color":"red"} }
									]
								}]
				}`),
			code: 201,
		},
		{
			input: []byte(`{
						"data":
								[{	"sku":"MS122-348",
										"productList":[
											{"productId": "886602023344", "metadata": {"size":"large"} }
										]
								}]
				}`),
			code: 201,
		},
	}

	mapp := Mapping{db, 1000}
	handler := web.Handler(mapp.PostSkuMapping)

	testHandlerHelper(JSONSample, handler, t)
}

func TestInsertMapping_InvalidCoeffs(t *testing.T) {
	db := dbSetup(t)
	var JSONSample = []inputTest{
		{
			input: []byte(`{
				"data":
					[{	"sku":"MS122-34",
						"productList":[
							{"productId": "886602023377", "dailyTurn": "helloworld", "metadata": {"color":"blue"} },
							{"productId": "888446970466", "dailyTurn": 0.1, "metadata": {"color":"red"} }
						]
					}]
			}`),
			code: 400,
		},
	}

	mapp := Mapping{db, 1000}
	handler := web.Handler(mapp.PostSkuMapping)

	testHandlerHelper(JSONSample, handler, t)
}

func TestInsertMapping_InvalidCoeffsMax(t *testing.T) {
	db := dbSetup(t)

	var JSONSample = []inputTest{
		{
			input: []byte(`{
				"data":
					[{	"sku":"MS122-34",
						"productList":[
							{"productId": "886602023377", "dailyTurn": 2.2, "metadata": {"color":"blue"} },
							{"productId": "888446970466", "dailyTurn": 0.1, "metadata": {"color":"red"} }
						]
					}]
				}`),
			code: 400,
		},
	}

	mapp := Mapping{db, 1000}
	handler := web.Handler(mapp.PostSkuMapping)

	testHandlerHelper(JSONSample, handler, t)
}

func TestInsertMapping_InvalidCoeffsMin(t *testing.T) {
	db := dbSetup(t)

	var JSONSample = []inputTest{
		{
			input: []byte(`{
						"data":
								[{	"sku":"MS122-34",
									"productList":[
										{"productId": "886602023377", "dailyTurn": 0.2, "metadata": {"color":"blue"} },
										{"productId": "888446970466", "dailyTurn": -0.1, "metadata": {"color":"red"} }
									]
								}]
				}`),
			code: 400,
		},
	}

	mapp := Mapping{db, 1000}
	handler := web.Handler(mapp.PostSkuMapping)

	testHandlerHelper(JSONSample, handler, t)
}

func TestInsertMapping_InvalidData(t *testing.T) {

	db := dbSetup(t)

	var invalidJSONSample = []inputTest{
		{
			input: []byte(`{
					"data":[{
							"ski":"MS122-34",
							"productList":[
								{"productId": "886602023377", "metadata": {"color":"blue"} },
								{"productId": "888446970466", "metadata": {"color":"red"} }
							]
					}]
				}`),
			code: 400,
		},
		{
			input: []byte(`{
					"data":[{
						"sku":"MS122-348",
						"productlist":[
							{"productId": "886602023344", "metadata": {"size":"large"} }
						]
					}]
				}`),
			code: 400,
		},
		{
			input: []byte(`{
					"data":[{
						"sku":"MS122-348",
						"productList":[
							{"productId": "886602023344", "metadata": {"size":"large"} }
							]}
						}]
				}`),
			code: 400,
		},
		{
			input: []byte(`{
					"data":[{
						"description":"black suede shoe size 9",
						{"productId": "886602023344", "metadata": {"size":"large"} }
					}
					}]
				}`),
			code: 400,
		},
		{
			input: []byte(`{
					"data":[{
						"sku":"MS122-348",
						"name":"BlackSuedeShoe",
						"description":"black suede shoe size 9",
						"title":"Les Miserables",
						"productList":[
							{"productId": "886602023344", "metadata": {"size":"large"} }
						]
					}]
				}`),
			code: 400,
		},
		{
			input: []byte(`{
					"data":[{
					"sku":"MS122-34",
					"name":"BlueSuedeShoe",
					"description":"blue suede shoe size 9",
					"productList":[
						"886602023377",
						"886602023377",
						"888446970466"
					]
					}]
				}`),
			code: 400,
		},

		{
			// Empty request body
			input: []byte(`{ }`),
			code:  400,
		},
	}

	mapp := Mapping{db, 1000}
	handler := web.Handler(mapp.PostSkuMapping)

	testHandlerHelper(invalidJSONSample, handler, t)
}

func TestGetProductBadRequest(t *testing.T) {
	urls := []string{
		"/productId/" + strings.Repeat("1", 2000),
		"/productId/asdf",
	}

	db := dbSetup(t)

	for _, url := range urls {
		request, err := http.NewRequest("GET", url, nil)
		if err != nil {
			t.Errorf("Unable to create new HTTP request %s", err.Error())
		}

		testRouter := mux.NewRouter().StrictSlash(true)
		testRecorder := httptest.NewRecorder()
		mapp := Mapping{db, config.AppConfig.ResponseLimit}
		testRouter.Path("/productId/{productId}").
			Name("testGetProductBadRequest").
			Handler(web.Handler(mapp.GetProductID))
		testRouter.ServeHTTP(testRecorder, request)

		if testRecorder.Code != http.StatusBadRequest {
			t.Errorf("Expected: %d Actual: %d\n%+v, %+v",
				http.StatusBadRequest, testRecorder.Code,
				testRecorder, request)
		}
	}
}

func TestGetProductID(t *testing.T) {
	db := dbSetup(t)
	insertSampleProductMetadata(db, t)

	testRouter := mux.NewRouter().StrictSlash(true)
	testRecorder := httptest.NewRecorder()
	mapp := Mapping{db, config.AppConfig.ResponseLimit}
	testHandler := web.Handler(mapp.GetProductID)
	testRouter.Path("/productid/{productId}").
		Name("testGetProductID").
		Handler(testHandler)

	url := "/productid/12345678912345"
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("Unable to create new HTTP request %+v", err)
	}
	testRouter.ServeHTTP(testRecorder, request)

	if testRecorder.Code != http.StatusOK {
		t.Errorf("Expected: %d; Actual: %d, %s\nrequest: %+v\nresponse: %+v",
			http.StatusOK, testRecorder.Code, testRecorder.Body,
			request, testRecorder)
	}
}

func TestGetProductIDBadRequestString(t *testing.T) {
	url := "/productid/00000000000000"

	db := dbSetup(t)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Errorf("Unable to create new HTTP request %s", err.Error())
	}
	mapp := Mapping{db, config.AppConfig.ResponseLimit}

	testRouter := mux.NewRouter().StrictSlash(true)

	testRecorder := httptest.NewRecorder()

	testHandler := web.Handler(mapp.GetProductID)

	testRouter.Path("/productid/{productId}").
		Name("testGetProductIDBadRequestString").
		Handler(testHandler)

	testRouter.ServeHTTP(testRecorder, request)

	if testRecorder.Code != http.StatusNotFound {
		t.Errorf("Expected: %d Actual: %d", http.StatusNotFound, testRecorder.Code)
	}
}

func insertSampleProductMetadata(db *sql.DB, t *testing.T) []productdata.SKUData {

	JSONSample := `[
		{ "sku":"MS122-33", "name":"mens formal pants",
		  "productList": [ {"productId": "12345678912345", "metadata": {"color":"blue"} } ]
		},
		{ "sku":"MS122-34", "name":"mens formal pants",
			"productList": [ {"productId": "12345678912346", "metadata": {"color":"blue"} } ]
		}
	]`

	var expectedMappings []productdata.SKUData
	err := json.Unmarshal([]byte(JSONSample), &expectedMappings)
	if err != nil {
		t.Fatalf("Not able to Unmarshal JSON object: %+v", err)
	}

	if err := productdata.Insert(db, expectedMappings); err != nil {
		t.Fatalf("Not able to insert into database: %+v", err)
	}

	return expectedMappings
}

func dbSetup(t *testing.T) *sql.DB {

	// Connect to PostgreSQL
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", config.AppConfig.DbHost,
		config.AppConfig.DbPort,
		config.AppConfig.DbUser, config.AppConfig.DbPass,
		config.AppConfig.DbName)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		t.Fatal(err)
	}
	// Create table
	db.Exec(productdata.DbSchema)

	return db
}
