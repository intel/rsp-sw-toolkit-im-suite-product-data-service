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
package web

import (
	"context"
	"net/http"

	"github.com/pborman/uuid"
)

// Key represents the type of value for the context key.
type ctxKey int

// KeyValues is how request values or stored/retrieved.
const KeyValues ctxKey = 1

// ContextValues used during log
type ContextValues struct {
	TraceID    string
	Method     string
	RequestURI string
}

// Handler is a type that handles a http request
type Handler func(context.Context, http.ResponseWriter, *http.Request) error

// A Middleware is a type that wraps a handler to remove boilerplate or other
// concerns not direct to any given Handler.
type Middleware func(Handler) Handler

// ServeHTTP interface is implemented by Handler
func (fn Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {

	// Create the context for the request.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	values := ContextValues{
		TraceID:    uuid.New(),
		Method:     request.Method,
		RequestURI: request.RequestURI,
	}
	ctx = context.WithValue(ctx, KeyValues, &values)

	// This function executes first on every request.
	if err := fn(ctx, writer, request); err != nil {
		// Respond with the error.
		Error(ctx, writer, err)
		return
	}

}
