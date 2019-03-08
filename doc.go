// Package main SKU Mapping API.
//
// Retailers typically refer to inventory based on SKU. This sku mapping service provides RRP with the necessary enterprise data to properly
// identify products in our system to notify Retailers of events that occur within RRP. Endpoints are provided for uploading SKU/UPC mapping
// data along with all associated metadata (ie. size, color, model) into the RRP System. RFID sensed inventory is read from item tags (EPCs)
// which then get converted to UPCs.
//
// __Known services this service depends on:__
//
// ○ Context Sensing
//
// ○ MongoDB
//
//
// __Known services that depend upon this service:__
//
// ○ Rules
//
// The is the schema associated to this URN urn:x-intel:context:thing:productmasterdata
//
//```json
// {
// "type": "urn:x-intel:context:thing:productmasterdata",
//"schema": {
//   "type": "object",
//   "properties": {
// 	"data": {
// 	  "type": "array",
// 	  "items": {
// 		"type": "object",
// 		"properties": {
// 		  "metadata": {
// 			"type": "object",
// 			"properties": {
// 			  "dept": {
// 				"type": "integer"
// 			  },
// 			  "supplier": {
// 				"type": "integer"
// 			  },
// 			  "sup_name": {
// 				"type": "string"
// 			  },
// 			  "class": {
// 				"type": "integer"
// 			  },
// 			  "vpn": {
// 				"type": "string"
// 			  },
// 			  "supp_color": {
// 				"type": "string"
// 			  },
// 			  "primary_upc_ind": {
// 				"type": "string"
// 			  },
// 			  "supp_size": {
// 				"type": "string"
// 			  },
// 			  "upc_desc": {
// 				"type": "string"
// 			  },
// 			  "unit_retail": {
// 				"type": "number"
// 			  }
// 			}
// 		  },
// 		  "dailyTurn":{
// 		  "type":["number"],
// 		  "minimum": 0,
// 		  "maximum": 1
// 		  },
// 		  "becomingReadable":{
// 		  "type":["number"],
// 		  "minimum": 0,
// 		  "maximum": 1
// 		  },
// 		  "exitError":{
// 		  "type":["number"],
// 		  "minimum": 0,
// 		  "maximum": 1
// 		  },
// 		  "beingRead":{
// 		  "type":["number"],
// 		  "minimum": 0,
// 		  "maximum": 1
// 		  },
// 		  "sku": {
// 			"type": "string"
// 		  },
// 		  "productId": {
// 			"type": "string"
// 		  }
// 		},
// 		"required": ["sku", "productId"]
// 	  }
// 	}
//   }
// },
// "descriptions": {
//   "en": {
// 	"documentation": "Reads the product masterdata from a csv file and publishes the data to the broker.",
// 	"short_name": "Product Master Data"
//   }
// }
// }
//```
//	## __Example configuration file json__
//```json
// {
//	&#9&#9"serviceName": "RRP - product-data-service",
//  &#9&#9"databaseName": "mapping",
//  &#9&#9"loggingLevel": "debug",
//  &#9&#9"secureMode" : false,
//  &#9&#9"skipCertVerify" : false,
//  &#9&#9"telemetryEndpoint": "http://166.130.9.122:8000",
//  &#9&#9"telemetryDataStoreName" : "Store105",
//  &#9&#9"port": "8080"
//  &#9&#9"responseLimit": 10000,
// }
//```
//	## __Example environment variables in compose File__:
//  ```json
//  {
//  &#9&#9contextSdk: "127.0.0.1:8888"
//  &#9&#9connectionString: "mongodb://127.0.0.1:27017"
//  }
// ```
// ###__Configuration file values__
// + `serviceName`  				 - Runtime name of the service
//
// + `databaseName`  				 - Name of database
//
// + `loggingLevel`  				 - Logging level to use: "info" (default) or "debug" (verbose)
//
// + `secureMode`  					 - Boolean flag indicating if using secure connection to the Context Brokers
//
// + `skipCertVerify`  				 - Boolean flag indicating if secure connection to the Context Brokers should skip certificate validation
//
// + `telemetryEndpoint`  				 - URL of the telemetry service receiving the metrics from the service
//
// + `telemetryDataStoreName`  		 - Name of the data store in the telemetry service to store the metrics
//
// + `port`  						 - Port to run the service/s HTTP Server on
//
// + `responseLimit`  				 - Default limit to what can be returned in a GET call - because of this, client must define their own top-skip functionality
//
// ###__Compose file environment variable values__
//
// + `contextSdk`  			- Host and port number for the Context Broker
//
// + `connectionString`  	- Host and port number for the Database connection
//
// ## __Known services this service depends on:__
// + context-broker
// These are the topics that this service subscribes to from the Context Sensing SDK Websocket bus. To learn more about the Context Sensing SDK, please visit http://contextsensing.intel.com/
// ```
//	&#9prodDataUrn        = "urn:x-intel:context:thing:productmasterdata"
//
// ```
// + rrp-mongo
//
// ## __Known services that depend upon this service:__
// + item-finder
// + rules-service
//
// Copyright 2018 Intel® Corporation, All rights reserved.
//
//     Schemes: http, https
//     Host: mapping-service:8080
//	   Contact:  RRP <rrp@intel.com>
//     BasePath: /
//     Version: 0.0.1
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
// swagger:meta
package main

// ProductID
//
// swagger:parameters productids
type productid struct {
	// a product id
	//
	// in: path
	ProductID string `json:"productId"`
}

// Sku
//
// swagger:parameters deleteSkus
type sku struct {
	// valid sku
	//
	// in: path
	Sku string `json:"sku"`
}

// Created
//
// swagger:response Created
type created struct {
}

// NotFound
//
// swagger:response NotFound
type notFound struct {
}
