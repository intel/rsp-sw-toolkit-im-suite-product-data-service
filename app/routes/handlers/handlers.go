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
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/xeipuuv/gojsonschema"
	"github.impcloud.net/RSP-Inventory-Suite/product-data-service/app/productdata"
	"github.impcloud.net/RSP-Inventory-Suite/product-data-service/pkg/web"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
)

// Mapping represents the User API method handler set.
type Mapping struct {
	MasterDB *sql.DB
	Size     int
}

// Response wraps results, inlinecount, and extra fields in a json object
// swagger:model resultsResponse
type Response struct {
	// Array containing results of query
	Results interface{} `json:"results"`
	Count   *int        `json:"count,omitempty"`
}

// ErrorList provides a collection of errors for processing
// swagger:response schemaValidation
type ErrorList struct {
	// The error list
	// in: body
	Errors []ErrReport `json:"errors"`
}

// ErrReport is used to wrap schema validation errors int json object
type ErrReport struct {
	Field       string      `json:"field"`
	ErrorType   string      `json:"errortype"`
	Value       interface{} `json:"value"`
	Description string      `json:"description"`
}

// Index is used for Docker Healthcheck commands to indicate
// whether the http server is up and running to take requests
// 200 OK
// nolint: unparam
func (mapp *Mapping) Index(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {
	web.Respond(ctx, writer, "Product Data Service", http.StatusOK)
	return nil
}

// GetSkuMapping retrieves sku mapping list
// 200 OK, 400 Bad Request, 500 Internal Error
func (mapp *Mapping) GetSkuMapping(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {

	results, count, err := productdata.Retrieve(mapp.MasterDB, request.URL.Query(), mapp.Size)
	if err != nil {
		return web.InvalidInputError(err)
	}

	// we need to check if this is a countType or if it's an array of interfaces
	if count != nil && results == nil {
		web.Respond(ctx, writer, count, http.StatusOK)
		return nil
	}

	resultSlice := reflect.ValueOf(results)

	if resultSlice.Len() < 1 {
		results = []productdata.SKUData{} // Return empty array
	}

	if count != nil && results != nil {
		web.Respond(ctx, writer, Response{Results: results, Count: &count.Count}, http.StatusOK)
		return nil
	}
	web.Respond(ctx, writer, Response{Results: results}, http.StatusOK)
	return nil
}

// PostSkuMapping maps SKU
// 200 OK, 500 Internal Error
func (mapp *Mapping) PostSkuMapping(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {

	mappings := productdata.Root{}

	// Reading request with a limit of 32mb
	body := make([]byte, request.ContentLength)
	_, err := io.ReadFull(request.Body, body)
	if err != nil {
		return err
	}

	// Validate JSON schema for mapping SKU service
	schemaLoader := gojsonschema.NewStringLoader(productdata.Schema)
	loader := gojsonschema.NewBytesLoader(body)

	// Validate schema
	result, err := gojsonschema.Validate(schemaLoader, loader)

	if err != nil {
		return web.InvalidInputError(err)
	}

	if !result.Valid() {
		errList := ErrorList{Errors: []ErrReport{}}
		for _, err := range result.Errors() {
			// err.Field() is not set for "required" error
			var field string
			if property, ok := err.Details()["property"].(string); ok {
				field = property
			} else {
				field = err.Field()
			}
			// ignore extraneous "number_one_of" error
			if err.Type() == "number_one_of" {
				continue
			}
			report := ErrReport{
				Description: err.Description(),
				Field:       field,
				ErrorType:   err.Type(),
				Value:       err.Value(),
			}
			errList.Errors = append(errList.Errors, report)
		}
		web.Respond(ctx, writer, errList, http.StatusBadRequest)
		return nil
	}

	if err := json.Unmarshal(body, &mappings); err != nil {
		return web.InvalidInputError(err)
	}

	if err := productdata.Insert(mapp.MasterDB, mappings.Data); err != nil {
		return err
	}

	web.Respond(ctx, writer, nil, http.StatusCreated)
	return nil
}

// GetProductID returns upc with metadata
// 200 OK, 400 Bad Request,  404 Not Found, 500 Internal Error
func (mapp *Mapping) GetProductID(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {

	metrics.GetOrRegisterGauge("Mapping-SKU.GetProductID.Attempt", nil).Update(1)
	startTime := time.Now()
	defer metrics.GetOrRegisterTimer("Mapping-SKU.GetProductID.Latency", nil).Update(time.Since(startTime))
	mSuccess := metrics.GetOrRegisterGauge("Mapping-SKU.GetProductID.Success", nil)
	mGetProductMetadataErr := metrics.GetOrRegisterGauge("Mapping-SKU.GetProductID.GetProductMetadataError", nil)

	vars := mux.Vars(request)
	productId := vars["productId"]

	if err := isValidProductID(productId); err != nil {
		web.Respond(ctx, writer, nil, http.StatusBadRequest)
		return err
	}

	prodData, err := productdata.GetProductMetadata(mapp.MasterDB, productId)
	if err != nil {
		if web.IsNotFoundError(err) {
			mGetProductMetadataErr.Update(1)
			web.Respond(ctx, writer, nil, http.StatusNotFound)
			return nil
		}
		return err
	}

	mSuccess.Update(1)
	web.Respond(ctx, writer, prodData.ProductList[0], http.StatusOK)
	return nil
}

func isValidProductID(productID string) error {
	if _, err := strconv.Atoi(productID); err != nil {
		return web.ValidationError("productID contains non integer characters")
	}
	if len(productID) < 1 || len(productID) > 1024 {
		return web.ValidationError("productID must be between 1 and 1024 characters")
	}
	return nil
}
