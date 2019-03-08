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
package middlewares

import (
	"context"
	"errors"
	"net/http"
	"runtime/debug"

	log "github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/product-data-service/pkg/web"
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
