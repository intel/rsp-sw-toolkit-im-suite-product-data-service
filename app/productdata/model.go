/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */
package productdata

// DbSchema postgresql db schema
const DbSchema = `
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS skus (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	data JSONB	
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sku
ON skus ((data->>'sku'));
`

// CountType is used to hold the total count
type CountType struct {
	Count int `json:"count"`
}

// Schema represents the schema for input data for RESTFul POST API
const Schema = `
{
    "definitions": {
        "productList": {
          "properties": {
            "productId": {
                "type": "string",
                "minLength": 1,
                "maxLength": 1024
            },
            "dailyTurn": {
                "type": "number",
                "minimum": 0,
                "maximum": 1
            },
            "becomingReadable": {
                "type": "number",
                "minimum": 0,
                "maximum": 1
            },
            "exitError": {
                "type": "number",
                "minimum": 0,
                "maximum": 1
            },
            "beingRead": {
                "type": "number",
                "minimum": 0,
                "maximum": 1
            },
            "metadata": {
            }
          },
          "additionalProperties": false,
          "type": "object"
        }
    },
    "type": "object",
    "required": [
        "data"
    ],
    "properties": {
        "data": {
            "type": "array",
            "minItems": 1,
            "items": {
                "type": "object",
                "oneOf": [
                    {
                        "required": [
                            "sku",
                            "productList"
                        ],
                        "properties": {
                            "sku": {
                                "type": "string"
                            },
                            "productList": {
                                "items": {
                                    "$ref": "#/definitions/productList"
                                },
                                "type": "array"
                            }
                        },
                        "additionalProperties": false
                    }
                ]
            }
        }
    },
    "additionalProperties": false
}
`

// IncomingData represents the struct of the raw data coming from the Broker.
//
// Although it may have the same "shape" as the ProductData, the json attributes
// may be different, but must be correctly mapped to the database model.
type IncomingData struct {
	// the Broker calls this `upc`, even though it could be any type of product ID
	ProductID        string                 `json:"upc"`
	SKU              string                 `json:"sku"`
	BeingRead        float64                `json:"beingRead"`
	BecomingReadable float64                `json:"becomingReadable"`
	ExitError        float64                `json:"exitError"`
	DailyTurn        float64                `json:"dailyTurn"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// SKUData connects a SKU to its list of products
type SKUData struct {
	// SKU represents an identifier a business uses for a product or collection of products
	SKU string `json:"sku" db:"sku"`
	// ProductList connects one or more products to the same SKU
	ProductList []ProductData `json:"productList" db:"productList"`
}

// ProductData models the product's attributes
type ProductData struct {
	// ProductID is the "formal" ID for a product, often a GTIN
	ProductID        string  `json:"productId" db:"productId"`
	BeingRead        float64 `json:"beingRead" db:"beingRead"`
	BecomingReadable float64 `json:"becomingReadable" db:"becomingReadable"`
	ExitError        float64 `json:"exitError" db:"exitError"`
	DailyTurn        float64 `json:"dailyTurn" db:"dailyTurn"`
	// Metadata stores arbitrary data about a product
	Metadata map[string]interface{} `json:"metadata"`
}

// Root - Main struct for input
// swagger:parameters postSkus
type Root struct {
	//in: body
	Data []SKUData `json:"data"`
}
