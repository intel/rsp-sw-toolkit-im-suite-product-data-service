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
package productdata

import (
	"net/url"
	"strconv"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	db "github.impcloud.net/RSP-Inventory-Suite/go-dbWrapper"
	odata "github.impcloud.net/RSP-Inventory-Suite/go-odata/mongo"
	"github.impcloud.net/RSP-Inventory-Suite/product-data-service/pkg/web"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
)

const productDataCollection = "skus"

// Write batch sizes must be between 1 and 1000. For safety, split it into 500 operations per call
// Because upsert requires a pair of upserting instructions. Use mongoMaxOps = 1000
const mongoMaxOps = 1000

// Retrieve gets the data out of the DB
func Retrieve(dbs *db.DB, query url.Values, maxSize int) (interface{}, *CountType, error) {
	// Metrics
	metrics.GetOrRegisterGauge(`Mapping-SKU.Retrieve.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Mapping-SKU.Retrieve.Success`, nil)
	mRetrieveErr := metrics.GetOrRegisterGauge("Mapping-SKU.Retrieve.Retrieve-Error", nil)
	mInputErr := metrics.GetOrRegisterGauge("Mapping-SKU.Retrieve.Input-Error", nil)
	mRetrieveLatency := metrics.GetOrRegisterTimer(`Mapping-SKU.Retrieve.Retrieve-Latency`, nil)

	count := query["$count"]

	// If count is true, and only $count is set return total count of the collection
	if len(count) > 0 && len(query) < 2 {
		return countHandler(dbs)
	}

	// Apply size limit if needed
	if len(query["$top"]) > 0 {

		topVal, err := strconv.Atoi(query["$top"][0])
		if err != nil {
			return nil, nil, web.ValidationError("invalid $top value")
		}

		if topVal > maxSize {
			query["$top"][0] = strconv.Itoa(maxSize)
		}

	} else {
		query["$top"] = []string{strconv.Itoa(maxSize)} // Apply size limit to the odata query
	}

	var object []interface{}
	// Else, run filter query and return slice of Mapping
	execFunc := func(collection *mgo.Collection) error {
		return odata.ODataQuery(query, &object, collection)
	}

	retrieveTimer := time.Now()
	if err := dbs.Execute(productDataCollection, execFunc); err != nil {
		if errors.Cause(err) == odata.ErrInvalidInput {
			mInputErr.Update(1)
			return nil, nil, web.InvalidInputError(err)
		}
		mRetrieveErr.Update(1)
		return nil, nil, errors.Wrap(err, "db.mapping.Find()")
	}
	mRetrieveLatency.Update(time.Since(retrieveTimer))

	// Check if inlinecount is set
	isInlineCount := query["$inlinecount"]

	if len(count) > 0 || (len(isInlineCount) > 0 && isInlineCount[0] == "allpages") {
		return inlineCountHandler(dbs, isInlineCount, object)
	}

	mSuccess.Update(1)
	return object, nil, nil

}

func countHandler(dbs *db.DB) (interface{}, *CountType, error) {

	mCountErr := metrics.GetOrRegisterGauge("Mapping-SKU.Retrieve.Count-Error", nil)
	mSuccess := metrics.GetOrRegisterGauge(`Mapping-SKU.Retrieve.Success`, nil)

	var count int
	var err error

	execFunc := func(collection *mgo.Collection) (int, error) {
		return odata.ODataCount(collection)
	}

	if count, err = dbs.ExecuteCount(productDataCollection, execFunc); err != nil {
		mCountErr.Update(1)
		return nil, nil, errors.Wrap(err, "db.mapping.Count()")
	}
	mSuccess.Update(1)
	return nil, &CountType{Count: count}, nil
}

func inlineCountHandler(dbs *db.DB, isInlineCount []string, object []interface{}) (interface{}, *CountType, error) {
	mCountErr := metrics.GetOrRegisterGauge("Mapping-SKU.Retrieve.Count-Error", nil)
	mSuccess := metrics.GetOrRegisterGauge(`Mapping-SKU.Retrieve.Success`, nil)
	var inlineCount int
	var err error

	// Get count from filtered data
	execInlineCount := func(collection *mgo.Collection) (int, error) {
		return odata.ODataInlineCount(collection)
	}

	if inlineCount, err = dbs.ExecuteCount(productDataCollection, execInlineCount); err != nil {
		mCountErr.Update(1)
		return nil, nil, errors.Wrap(err, "db.mapping.Count()")
	}

	// if $inlinecount is set, return results and inlinecount
	if len(isInlineCount) > 0 {

		if isInlineCount[0] == "allpages" {

			return object, &CountType{Count: inlineCount}, nil
		}
	}

	// if $count is set with $filter, return only the count of the filtered results
	mSuccess.Update(1)
	return nil, &CountType{Count: inlineCount}, nil

}

func findAndUpdateSkus(dbs *db.DB, skuData *[]SKUData) error {

	var skusList []string
	for _, sku := range *skuData {
		skusList = append(skusList, sku.SKU)
	}

	var results []SKUData
	execFunc := func(collection *mgo.Collection) error {
		return collection.Find(bson.M{"sku": bson.M{"$in": skusList}}).All(&results)
	}

	if err := dbs.Execute(productDataCollection, execFunc); err != nil {
		return err
	}

	mergeProductList(skuData, &results)

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
// In case of slice greater than 500 elements, Insert will use a bulk operation in batch of 500
func Insert(dbs *db.DB, skuData []SKUData) error {

	// Metrics
	metrics.GetOrRegisterGauge(`Mapping-SKU.Insert.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Mapping-SKU.Insert.Success`, nil)
	mInsertErr := metrics.GetOrRegisterGauge("Mapping-SKU.Insert.Insert-Error", nil)
	mInsertLatency := metrics.GetOrRegisterTimer(`Mapping-SKU.Insert.Insert-Latency`, nil)
	mSkuInsertCount := metrics.GetOrRegisterGaugeCollection("Mapping-SKU.Insert.Count", nil)

	// TODO: Consider a total bytes processed metric for this function.  Check with dev team.

	startTime := time.Now()

	//Create Bulk upsert interface input
	skus := make([]interface{}, len(skuData)*2)

	// Validate and prepare data into pairs of key,obj
	keyPairs := 0

	// Find and merge product list with existing data in db
	if err := findAndUpdateSkus(dbs, &skuData); err != nil {
		return err
	}

	for _, item := range skuData {
		// Validate empty sku or productList
		if item.SKU == "" || len(item.ProductList) == 0 {
			return web.ValidationError(
				"Unable to insert empty SKUs or Product ID attributes")
		}

		// Remove duplicate product IDs, if any
		item.ProductList = removeDuplicateProducts(item.ProductList)

		// Upsert requires a pair of upserting instructions (select, obj)
		// e.g. ["key",obj,"key2",obj2,"key3", obj3]
		skus[keyPairs] = bson.M{"sku": item.SKU}
		skus[keyPairs+1] = item
		keyPairs += 2
	}

	bulkFunc := func(collection *mgo.Collection) *mgo.Bulk {
		return collection.Bulk()
	}

	bulk := dbs.ExecuteBulk(productDataCollection, bulkFunc)
	bulk.Unordered()

	// Upsert in batch of 500 due to mongodb 1000 max ops limitation
	// 1000 because is a pair of instructions. Thus, 500 items means 1000 size
	if len(skus) > mongoMaxOps {
		if err := bulkOperation(skus, dbs, bulk, bulkFunc); err != nil {
			mInsertErr.Update(1)
			return err
		}

	} else {
		bulk.Upsert(skus...)
		if _, err := bulk.Run(); err != nil {
			mInsertErr.Update(1)
			return errors.Wrap(err, "Unable to insert SKUs in database (db.bulk.upsert)")
		}
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

func bulkOperation(skus []interface{}, dbs *db.DB, bulk *mgo.Bulk, bulkFunc func(collection *mgo.Collection) *mgo.Bulk) error {

	range1 := 0
	range2 := mongoMaxOps
	lastBatch := false

	for {

		// Queue batches of 1000 elements which translates to 500 operations
		/*
			TODO: Check with the team on if we want to count each bulk batch upsert success or just success of the entire upsert?
			I assume that a single success would suffice.
		*/
		if range2 < len(skus) {
			bulk.Upsert(skus[range1:range2]...)
		} else {
			// Last batch
			bulk.Upsert(skus[range1:]...)
			lastBatch = true
		}

		if _, err := bulk.Run(); err != nil {
			return errors.Wrap(err, "Unable to insert SKUs in database (db.bulk.upsert)")
		}

		// Flush any queued data
		// Reinitialize bulk after being flushed
		bulk = nil
		bulk = dbs.ExecuteBulk(productDataCollection, bulkFunc)
		bulk.Unordered()

		// Break after last batch
		if lastBatch {
			break
		}
		range1 = range2
		range2 += mongoMaxOps

	}

	return nil
}

// GetProductMetadata receives a product ID and looks up and returns the corresponding metadata
func GetProductMetadata(dbs *db.DB, productId string) (SKUData, error) {

	metrics.GetOrRegisterGauge("Mapping-SKU.GetProductMetadata.Attempt", nil).Update(1)
	startTime := time.Now()
	defer metrics.GetOrRegisterTimer("Mapping-SKU.GetProductMetadata.Latency", nil).Update(time.Since(startTime))
	mSuccess := metrics.GetOrRegisterGauge("Mapping-SKU.GetProductMetadata.Success", nil)
	mMongoErr := metrics.GetOrRegisterGauge("Mapping-SKU.GetProductMetadata.MongoError", nil)

	var skuData SKUData

	execFunc := func(collection *mgo.Collection) error {
		return collection.Find(bson.M{"productList.productId": productId}).
			Select(bson.M{"productList": bson.M{
				"$elemMatch": bson.M{
					"productId": productId}}}).One(&skuData)
	}
	if err := dbs.Execute(productDataCollection, execFunc); err != nil {
		if err == mgo.ErrNotFound {
			mSuccess.Update(1)
			return SKUData{}, web.NotFoundError()
		}
		mMongoErr.Update(1)
		return SKUData{}, errors.Wrap(err, "db.mapping.find()")
	}

	mSuccess.Update(1)
	return skuData, nil
}

// Delete -- Deletes a given SKU mapping from the database
func Delete(dbs *db.DB, sku string) error {
	metrics.GetOrRegisterGauge(`Mapping-SKU.Delete.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`Mapping-SKU.Delete.Success`, nil)
	mDeleteErr := metrics.GetOrRegisterGauge("Mapping-SKU.Delete.Delete-Error", nil)

	execFunc := func(collection *mgo.Collection) error {
		return collection.Remove(bson.M{"sku": sku})
	}
	if err := dbs.Execute(productDataCollection, execFunc); err != nil {
		if err == mgo.ErrNotFound {
			mSuccess.Update(1)
			return web.NotFoundError()
		}
		mDeleteErr.Update(1)
		return errors.Wrap(err, "db.mapping.Delete()")
	}
	mSuccess.Update(1)
	return nil
}
