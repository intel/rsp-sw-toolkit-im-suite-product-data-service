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
package productdata

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"
	odata "github.impcloud.net/RSP-Inventory-Suite/go-odata/postgresql"
	"github.impcloud.net/RSP-Inventory-Suite/product-data-service/pkg/web"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
)

const productDataTable = "skus"
const jsonbColumn = "data"

type prodDataWrapper struct {
	ID   []uint8 `db:"id" json:"id"`
	Data SKUData `db:"data" json:"data"`
}

// Retrieve gets the data out of the DB
func Retrieve(db *sql.DB, query url.Values, maxSize int) ([]SKUData, *CountType, error) {

	// Metrics
	metrics.GetOrRegisterGauge(`Product-Data.Retrieve.Attempt`, nil).Update(1)
	mCountErr := metrics.GetOrRegisterGauge("Product-Data.Retrieve.Count-Error", nil)
	mSuccess := metrics.GetOrRegisterGauge(`Product-Data.Retrieve.Success`, nil)
	mRetrieveErr := metrics.GetOrRegisterGauge("Product-Data.Retrieve.Retrieve-Error", nil)
	mInputErr := metrics.GetOrRegisterGauge("Product-Data.Retrieve.Input-Error", nil)
	mRetrieveLatency := metrics.GetOrRegisterTimer(`Product-Data.Retrieve.Retrieve-Latency`, nil)

	if db == nil {
		return nil, nil, errors.New("No database connection")
	}

	countQuery := query["$count"]

	// If only $count is set, return total count of the table
	if len(countQuery) > 0 && len(query) == 1 {

		var count int

		row := db.QueryRow("SELECT count(*) FROM skus")
		err := row.Scan(&count)
		if err != nil {
			mCountErr.Update(1)
			return []SKUData{}, nil, err
		}

		mSuccess.Update(1)
		return []SKUData{}, &CountType{Count: count}, nil
	}

	// Apply size limit if needed
	if len(query["$top"]) > 0 {

		topVal, err := strconv.Atoi(query["$top"][0])
		if err != nil {
			return []SKUData{}, nil, web.ValidationError("invalid $top value")
		}

		if topVal > maxSize {
			query["$top"][0] = strconv.Itoa(maxSize)
		}

	} else {
		query["$top"] = []string{strconv.Itoa(maxSize)} // Apply size limit to the odata query
	}

	// Else, run filter query and return slice of SKUData
	retrieveTimer := time.Now()

	// Run OData PostgreSQL
	rows, err := odata.ODataSQLQuery(query, productDataTable, jsonbColumn, db)
	if err != nil {
		if errors.Cause(err) == odata.ErrInvalidInput {
			mInputErr.Update(1)
			return []SKUData{}, nil, web.InvalidInputError(err)
		}
		return []SKUData{}, nil, errors.Wrap(err, "db.Select")
	}

	defer rows.Close()

	prodSlice := make([]SKUData, 0)

	// Loop through the results and append them to a slice
	for rows.Next() {

		prodDataWrapper := new(prodDataWrapper)
		err := rows.Scan(&prodDataWrapper.ID, &prodDataWrapper.Data)
		if err != nil {
			mRetrieveErr.Update(1)
			return []SKUData{}, nil, err
		}
		prodSlice = append(prodSlice, prodDataWrapper.Data)

	}
	if err = rows.Err(); err != nil {
		mRetrieveErr.Update(1)
		return []SKUData{}, nil, err
	}
	mRetrieveLatency.Update(time.Since(retrieveTimer))

	// Check if $inlinecount or $count is set in combination with $filter
	isInlineCount := query["$inlinecount"]

	if len(isInlineCount) > 0 && isInlineCount[0] == "allpages" {
		mSuccess.Update(1)
		return prodSlice, &CountType{Count: len(prodSlice)}, nil
	} else if len(countQuery) > 0 {
		mSuccess.Update(1)
		return []SKUData{}, &CountType{Count: len(prodSlice)}, nil
	}

	mSuccess.Update(1)
	return prodSlice, nil, nil

}

// Value implements driver.Valuer inferfaces
func (s SKUData) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan implements sql.Scanner inferfaces
func (s *SKUData) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &s)
}

func findAndUpdateSkus(db *sql.DB, skuData *[]SKUData) error {

	var skusList strings.Builder
	for _, sku := range *skuData {
		skusList.WriteString(pq.QuoteLiteral(sku.SKU))
	}

	selectQuery := fmt.Sprintf("SELECT %s FROM %s WHERE %s ->> 'sku' IN ($1)",
		pq.QuoteIdentifier(jsonbColumn),
		pq.QuoteIdentifier(productDataTable),
		pq.QuoteIdentifier(jsonbColumn))

	rows, err := db.Query(selectQuery, skusList.String())
	if err != nil {
		return err
	}
	defer rows.Close()

	prodSlice := make([]SKUData, 0)

	for rows.Next() {

		prodDataWrapper := new(prodDataWrapper)
		err := rows.Scan(&prodDataWrapper.Data)
		if err != nil {
			return err
		}
		prodSlice = append(prodSlice, prodDataWrapper.Data)

	}
	if err = rows.Err(); err != nil {
		return err
	}

	mergeProductList(skuData, &prodSlice)

	return nil
}

func mergeProductList(incoming *[]SKUData, current *[]SKUData) {

	currentMap := make(map[string]SKUData, len(*current))

	for _, item := range *current {
		currentMap[item.SKU] = item
	}

	for incomingIndex := range *incoming {

		currentSku, ok := currentMap[(*incoming)[incomingIndex].SKU]
		if !ok {
			continue
		}

		var newProductList []ProductData

		for _, product := range (*incoming)[incomingIndex].ProductList {
			found := false
			// Merge ProductIDs eliminating duplicates
			for _, currentProduct := range currentSku.ProductList {
				if product.ProductID == currentProduct.ProductID {
					currentProduct.Metadata = product.Metadata
					currentProduct.DailyTurn = product.DailyTurn
					currentProduct.BecomingReadable = product.BecomingReadable
					currentProduct.BeingRead = product.BeingRead
					currentProduct.ExitError = product.ExitError
					newProductList = append(newProductList, currentProduct)
					found = true
					break
				}
			}

			if !found {
				newProductList = append(newProductList, product)
			}
		}

		(*incoming)[incomingIndex].ProductList = newProductList
	}
}

// Insert receives a slice of sku mapping and inserts them to the database
func Insert(db *sql.DB, skuData []SKUData) error {

	// Metrics
	metrics.GetOrRegisterGauge(`Product-Data.Insert.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Product-Data.Insert.Success`, nil)
	mInsertErr := metrics.GetOrRegisterGauge("Product-Data.Insert.Insert-Error", nil)
	mInsertLatency := metrics.GetOrRegisterTimer(`Product-Data.Insert.Insert-Latency`, nil)
	mSkuInsertCount := metrics.GetOrRegisterGaugeCollection("Product-Data.Insert.Count", nil)

	// TODO: Consider a total bytes processed metric for this function.  Check with dev team.

	startTime := time.Now()

	//Create Bulk upsert interface input
	skus := make([]interface{}, len(skuData)*2)

	// Find and merge product list with existing data in db
	if err := findAndUpdateSkus(db, &skuData); err != nil {
		return err
	}

	var upsertStmt strings.Builder

	for _, item := range skuData {

		// Validate empty sku or productList
		if item.SKU == "" || len(item.ProductList) == 0 {
			return web.ValidationError(
				"Unable to insert empty SKUs or Product ID attributes")
		}

		// Remove duplicate product IDs, if any
		item.ProductList = removeDuplicateProducts(item.ProductList)

		obj, err := json.Marshal(item)
		if err != nil {
			return err
		}

		// Example of Upsert:
		//  INSERT INTO skus (data) VALUES ('{ "sku":"MS122-33", "name":"mens formal pants",
		//  "productList": [ {"productId": "12345678912345", "metadata": {"color":"blue"} } ]
		//  }')
		//  ON CONFLICT (( data  ->> 'sku' ))
		//  DO UPDATE SET data = sku.data || '{ "sku":"MS122-33", "name":"mens formal pants",
		//  "productList": [ {"productId": "12345678912345", "metadata": {"color":"red"} } ]
		//  }';

		upsertClause := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) 
									 ON CONFLICT (( %s  ->> 'sku' )) 
									 DO UPDATE SET %s = %s.%s || %s; `,
			pq.QuoteIdentifier(productDataTable),
			pq.QuoteIdentifier(jsonbColumn),
			pq.QuoteLiteral(string(obj)),
			pq.QuoteIdentifier(jsonbColumn),
			pq.QuoteIdentifier(jsonbColumn),
			pq.QuoteIdentifier(productDataTable),
			pq.QuoteIdentifier(jsonbColumn),
			pq.QuoteLiteral(string(obj)),
		)

		// Making all upsert sql statements into one network call
		upsertStmt.WriteString(upsertClause)

	}

	// Not able to implement transactions since they do not support multiple sql statements
	_, err := db.Exec(upsertStmt.String())
	if err != nil {
		mInsertErr.Update(1)
		return err
	}

	mSkuInsertCount.Add(int64(len(skus)))
	mInsertLatency.Update(time.Since(startTime))
	mSuccess.Update(1)
	return nil
}

func removeDuplicateProducts(productItems []ProductData) []ProductData {

	productMap := make(map[string]bool)
	var newProductList []ProductData

	for _, item := range productItems {

		if _, ok := productMap[item.ProductID]; !ok {
			productMap[item.ProductID] = true
			newProductList = append(newProductList, item)
		}
	}

	return newProductList

}

// GetProductMetadata receives a product ID (upc) and looks up and returns the corresponding metadata
func GetProductMetadata(db *sql.DB, productID string) (SKUData, error) {

	metrics.GetOrRegisterGauge("Product-Data.GetProductMetadata.Attempt", nil).Update(1)
	startTime := time.Now()
	defer metrics.GetOrRegisterTimer("Product-Data.GetProductMetadata.Latency", nil).Update(time.Since(startTime))
	mSuccess := metrics.GetOrRegisterGauge("Product-Data.GetProductMetadata.Success", nil)
	mDbErr := metrics.GetOrRegisterGauge("Product-Data.GetProductMetadata.DbError", nil)

	var skuData SKUData

	selectQuery := fmt.Sprintf(`SELECT %s FROM %s WHERE %s -> 'productList' @> '[{"productId": %s }]' LIMIT 1`,
		pq.QuoteIdentifier(jsonbColumn),
		pq.QuoteIdentifier(productDataTable),
		pq.QuoteIdentifier(jsonbColumn),
		pq.QuoteIdentifier(productID),
	)

	if err := db.QueryRow(selectQuery).Scan(&skuData); err != nil {

		if err == sql.ErrNoRows {
			mSuccess.Update(1)
			return SKUData{}, web.NotFoundError()
		}

		mDbErr.Update(1)
		return SKUData{}, err
	}

	mSuccess.Update(1)
	return skuData, nil
}
