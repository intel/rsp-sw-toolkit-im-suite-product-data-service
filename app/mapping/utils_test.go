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
