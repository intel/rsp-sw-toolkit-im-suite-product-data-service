/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */
package productdata

// GroupBySku creates a map of SKUData with sku as key and groups same sku object.
func GroupBySku(prodDataSlice []SKUData) map[string][]SKUData {

	prodDataMap := make(map[string][]SKUData, len(prodDataSlice))

	for _, value := range prodDataSlice {
		prodDataMap[value.SKU] = append(prodDataMap[value.SKU], value)
	}

	return prodDataMap
}

// MapBySku appends Product and Metadata to ProductList array
func MapBySku(prodDataMap map[string][]SKUData) []SKUData {
	var results []SKUData

	for sku, skuList := range prodDataMap {
		mergedSKU := SKUData{SKU: sku}

		for _, otherSKU := range skuList {
			var productData ProductData

			if len(otherSKU.ProductList) > 0 {
				productData.ProductID = otherSKU.ProductList[0].ProductID
				productData.BeingRead = otherSKU.ProductList[0].BeingRead
				productData.BecomingReadable = otherSKU.ProductList[0].BecomingReadable
				productData.DailyTurn = otherSKU.ProductList[0].DailyTurn
				productData.ExitError = otherSKU.ProductList[0].ExitError
				productData.Metadata = otherSKU.ProductList[0].Metadata
			}

			mergedSKU.ProductList = append(mergedSKU.ProductList, productData)
		}

		results = append(results, mergedSKU)
	}

	return results
}
