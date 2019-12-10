/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */
package productdata

import (
	"reflect"
	"testing"
)

func TestGroupBy(t *testing.T) {

	prodMockData := make([]SKUData, 3)

	skuObj1 := SKUData{SKU: "123", ProductList: []ProductData{{ProductID: "1234"}}}
	skuObj2 := SKUData{SKU: "123", ProductList: []ProductData{{ProductID: "12345"}}}
	skuObj3 := SKUData{SKU: "1234", ProductList: []ProductData{{ProductID: "123456"}}}

	prodMockData[0] = skuObj1
	prodMockData[1] = skuObj2
	prodMockData[2] = skuObj3

	// Actual result
	expectedResult := make(map[string][]SKUData, 2)
	expectedResult["123"] = append(expectedResult["123"], SKUData{SKU: "123", ProductList: []ProductData{{ProductID: "1234"}}})
	expectedResult["123"] = append(expectedResult["123"], SKUData{SKU: "123", ProductList: []ProductData{{ProductID: "12345"}}})
	expectedResult["1234"] = append(expectedResult["1234"], SKUData{SKU: "1234", ProductList: []ProductData{{ProductID: "123456"}}})

	actualResult := GroupBySku(prodMockData)

	// Verify if actual Result is equal to expected Result
	if !reflect.DeepEqual(actualResult, expectedResult) {
		t.Error("actual result is not equal to expected result")
	}

}

func TestMapBySku(t *testing.T) {

	prodDataMap := make(map[string][]SKUData, 2)
	prodDataMap["123"] = append(prodDataMap["123"], SKUData{SKU: "123", ProductList: []ProductData{{ProductID: "1234"}}})
	prodDataMap["123"] = append(prodDataMap["123"], SKUData{SKU: "123", ProductList: []ProductData{{ProductID: "12345"}}})
	prodDataMap["1234"] = append(prodDataMap["1234"], SKUData{SKU: "1234", ProductList: []ProductData{{ProductID: "123456"}}})

	// Actual Result
	expectedResult := make([]SKUData, 2)
	expectedResult[0] = SKUData{SKU: "123", ProductList: []ProductData{{ProductID: "1234"}, {ProductID: "12345"}}}
	expectedResult[1] = SKUData{SKU: "1234", ProductList: []ProductData{{ProductID: "123456"}}}

	actualResult := MapBySku(prodDataMap)

	if !compareProdDataSlices(actualResult, expectedResult) {
		t.Error("actual result is not equal to expected result")
	}
}

func compareProdDataSlices(prodData1 []SKUData, prodData2 []SKUData) bool {

	if len(prodData1) != len(prodData2) {
		return false
	}

	for _, value := range prodData1 {

		if !containsSku(value.SKU, prodData2) {
			return false
		}

		for _, item := range value.ProductList {

			if !containsProductID(value.SKU, item.ProductID, prodData2) {
				return false
			}

		}

	}

	return true

}

func containsSku(sku string, prodData []SKUData) bool {

	for _, value := range prodData {
		if value.SKU == sku {
			return true
		}
	}

	return false
}

func containsProductID(sku string, productID string, prodData []SKUData) bool {

	for _, value := range prodData {
		if value.SKU == sku {
			for _, productIDs := range value.ProductList {
				if productIDs.ProductID == productID {
					return true
				}
			}
		}
	}
	return false
}
