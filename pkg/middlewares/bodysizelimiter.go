/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */
package middlewares

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/intel/rsp-sw-toolkit-im-suite-product-data-service/pkg/web"
	log "github.com/sirupsen/logrus"
)

// max size limit of body 16MB
const (
	requestMaxSize = 16 << 20
)

// BodyLimiter middleware
func BodyLimiter(next web.Handler) web.Handler {
	return web.Handler(func(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {
		if request.Method == http.MethodPost || request.Method == http.MethodPut {
			tracerID := ctx.Value(web.KeyValues).(*web.ContextValues).TraceID

			// check based on content length
			headerSet := request.Header.Get("Content-Length")
			if headerSet != "" && request.ContentLength > requestMaxSize {
				log.WithFields(log.Fields{
					"Method":     request.Method,
					"RequestURI": request.RequestURI,
					"TraceID":    tracerID,
					"Code":       http.StatusRequestEntityTooLarge,
				}).Error("Request entity too large")
				return web.EntityTooLargeError()
			}

			// If header not set, set content length based on actual size of the body
			if headerSet == "" {
				var buf bytes.Buffer
				reqBody := http.MaxBytesReader(writer, request.Body, requestMaxSize)
				bodySize, err := buf.ReadFrom(reqBody)
				if err != nil {
					log.WithFields(log.Fields{
						"Method":     request.Method,
						"RequestURI": request.RequestURI,
						"TraceID":    tracerID,
						"Code":       http.StatusRequestEntityTooLarge,
					}).Error("Request entity too large")
					return web.EntityTooLargeError()
				}
				request.Header.Set("Content-Length", strconv.Itoa(int(bodySize)))
				request.Body = ioutil.NopCloser(&buf)
			}

		}
		return next(ctx, writer, request)
	})

}
