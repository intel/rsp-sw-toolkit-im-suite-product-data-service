/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */
package middlewares

import (
	"context"
	"errors"
	"net/http"
	"runtime/debug"

	"github.com/intel/rsp-sw-toolkit-im-suite-product-data-service/pkg/web"
	log "github.com/sirupsen/logrus"
)

// Recover middleware
func Recover(next web.Handler) web.Handler {
	return web.Handler(func(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {
		// Recover from any panic
		defer func() {
			if r := recover(); r != nil {
				traceID := ctx.Value(web.KeyValues).(*web.ContextValues).TraceID

				log.WithFields(log.Fields{
					"Method":     request.Method,
					"RequestURI": request.RequestURI,
					"TraceID":    traceID,
					"Code":       http.StatusInternalServerError,
					"Error":      r,
					"Stacktrace": string(debug.Stack()),
				}).Error("Panic Caught")

				web.RespondError(ctx, writer, errors.New("an error has occurred"), http.StatusInternalServerError)
			}
		}()
		// Go to the next http handler
		err := next(ctx, writer, request)
		return err
	})
}
