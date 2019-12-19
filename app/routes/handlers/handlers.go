/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/intel/rsp-sw-toolkit-im-suite-utilities/go-metrics"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/intel/rsp-sw-toolkit-im-suite-gojsonschema"
	"github.com/intel/rsp-sw-toolkit-im-suite-product-data-service/app/productdata"
	"github.com/intel/rsp-sw-toolkit-im-suite-product-data-service/pkg/web"
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
	if count != nil && len(results) == 0 {
		web.Respond(ctx, writer, count, http.StatusOK)
		return nil
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

	metrics.GetOrRegisterGauge("Product-Data.GetProductID.Attempt", nil).Update(1)
	startTime := time.Now()
	defer metrics.GetOrRegisterTimer("Product-Data.GetProductID.Latency", nil).Update(time.Since(startTime))
	mSuccess := metrics.GetOrRegisterGauge("Product-Data.GetProductID.Success", nil)
	mGetProductMetadataErr := metrics.GetOrRegisterGauge("Product-Data.GetProductID.GetProductMetadataError", nil)

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
