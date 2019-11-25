/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
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
