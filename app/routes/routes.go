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
package routes

import (
	"net/http"

	"github.com/gorilla/mux"

	db "github.impcloud.net/Responsive-Retail-Core/mongodb"
	"github.impcloud.net/RSP-Inventory-Suite/product-data-service/app/routes/handlers"
	"github.impcloud.net/RSP-Inventory-Suite/product-data-service/pkg/middlewares"
	"github.impcloud.net/RSP-Inventory-Suite/product-data-service/pkg/web"
)

// Route struct holds attributes to declare routes
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc web.Handler
}

// NewRouter creates the routes for GET and POST
func NewRouter(masterDB *db.DB, size int) *mux.Router {

	mapp := handlers.Mapping{MasterDB: masterDB, Size: size}

	var routes = []Route{
		// swagger:operation GET / default Healthcheck
		//
		// Healthcheck Endpoint
		//
		// Endpoint that is used to determine if the application is ready to take web requests
		//
		// ---
		// consumes:
		// - application/json
		//
		// produces:
		// - application/json
		//
		//
		// schemes:
		// - http
		//
		// responses:
		//   '200':
		//     description: OK
		//
		{
			"Index",
			"GET",
			"/",
			mapp.Index,
		},
		// swagger:route POST /skus skus postSkus
		//
		// Loads SKU Data
		//
		// This API call is used to upload a list of SKU items into Responsive Retail.<br>
		// The SKU data includes:
		//
		// <blockquote>• <b>SKU</b>: The SKU number, a unique identifier for the SKU.</blockquote>
		//
		// <blockquote>• <b>UPC List</b>: A list (array) of UPCs that are included in the SKU.</blockquote>
		//
		// Expected formatting of JSON input (as an example):<br><br>
		//
		//```json
		// {
		//"data":[{
		//   "sku" : "MS122-32",
		//   "productList" : [
		//     { "upc": "00888446671444", "metadata": {"color":"blue"} },
		//     { "upc": "889319762751", "metadata": {"size":"small"} }
		//    ]
		// 	 },
		// 	{
		//    "sku" : "MS122-34",
		//    "upcList" : [
		//      {"upc": "90388987132758", "metadata": {"name":"pants"} }
		//    ]
		//  }]
		// }
		//```
		// <br>
		// Each SKU item is treated individually; it succeeds or fails independent of the other SKUs.
		// Check the returned results to determine the success or failure of each SKU.
		//
		//     Consumes:
		//     - application/json
		//
		//     Produces:
		//     - application/json
		//
		//     Schemes: http
		//
		//
		//     Responses:
		//       201: Created
		//       400: schemaValidation
		//       500: internalError
		//
		{
			"PostSkuMapping",
			"POST",
			"/skus",
			mapp.PostSkuMapping,
		},
		// swagger:route GET /skus skus getSkus
		//
		// Retrieves SKU Data
		//
		// This API call is used to retrieve a list of SKU items.
		//
		// <blockquote>• <b>Search by sku</b>: To search by sku, you would use the filter query parameter like so: /sku?$filter=(sku eq 'MS122-32')</blockquote>
		//
		// <blockquote>• <b>Search by name</b>: To search by name, you would use the filter query parameter like so: /location?$filter=(name eq 'mens khaki slacks')</blockquote>
		//
		//
		// `/skus?$top=10&$select=sku` - Useful for paging data. Grab the top 10 records and only pull back the sku field
		//
		// `/skus?$count` - Tell me how many records are in the database
		//
		// `/skus?$filter=(sku eq '12345678') and (upclist.metadata.color eq 'red')` - This filters on particular sku and UPCs that are classified as "Red"
		//
		// `/skus?$orderby=sku desc` - Give me back all skus in descending order by sku
		//
		// `/skus?$filter=startswith(sku,'m')` - Give me all skus that begin with the letter 'm'
		//
		// `/skus?$count&$filter=(sku eq '12345678')` - Give me the count of items with the SKU `12345678``
		//
		// `/skus?$inlinecount=allpages&$filter=(sku eq '12345678')` - Give me all items with the SKU `12345678` and include how many there are
		//
		//
		//
		// Example Result:<br><br>
		//```json
		// {
		//     "results": [
		//         {
		//             "sku": "12345679",
		//             "upclist": [
		//                 {
		//                     "metadata": {
		//                         "color": "blue",
		//                         "size": "XS"
		//                     },
		//                     "upc": "123456789783"
		//                 },
		//                 {
		//                     "metadata": {
		//                         "color": "red",
		//                         "size": "M"
		//                     },
		//                     "upc": "123456789784"
		//                 }
		//             ]
		//         }
		//     ]
		// }
		//```
		//
		//     Consumes:
		//     - application/json
		//
		//     Produces:
		//     - application/json
		//
		//     Schemes: http
		//
		//     Responses:
		//       200: body:resultsResponse
		//       400: schemaValidation
		//       500: internalError
		//
		{
			"GetSkuMapping",
			"GET",
			"/skus",
			mapp.GetSkuMapping,
		},
		// swagger:route GET /productid/{productid} productid productids
		//
		// Retrieves SKU Data
		//
		// This API call is used to get the metadata for a upc.<br><br>
		//
		// Example query:
		//
		// <blockquote>/upc/12345678978345</blockquote> <br><br>
		//
		//
		// Example Result: <br><br>
		//```json
		// {
		//   "metadata": {
		// 				"color": "blue",
		// 				 "size": "XS"
		// 			  },
		//   "upc": "12345678978345"
		// }
		//```
		//
		//     Consumes:
		//     - application/json
		//
		//     Produces:
		//     - application/json
		//
		//     Schemes: http
		//
		//     Responses:
		//       200: body:resultsResponse
		//       404: NotFound
		//       400: schemaValidation
		//       500: internalError
		//
		{
			"GetProductID",
			"GET",
			"/productid/{productId}",
			mapp.GetProductID,
		},
		// swagger:route DELETE /skus/{sku} skus deleteSkus
		//
		// Deletes SKU data
		//
		// This API call is used to delete a sku object with its upcList<br><br>
		//
		// Example query:
		//
		// <blockquote>/skus/MS122-32</blockquote> <br><br>
		//
		//     Consumes:
		//     - application/json
		//
		//     Produces:
		//     - application/json
		//
		//     Schemes: http
		//
		//     Responses:
		//       204: body:resultsResponse
		//       404: NotFound
		//       500: internalError
		//
		{
			"DeleteSku",
			http.MethodDelete,
			"/skus/{sku}",
			mapp.DeleteSku,
		},
	}

	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {

		handler := route.HandlerFunc
		handler = middlewares.Recover(handler)
		handler = middlewares.Logger(handler)
		handler = middlewares.BodyLimiter(handler)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	return router
}
