/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */
package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// JSONError is the response for errors that occur within the API.
// swagger:response internalError
type JSONError struct {
	// The error message
	// in: body
	Error string `json:"error"`
}

// CommonError makes it easier to identify and handle common errors.
type CommonError struct {
	error
	Code int
}

// NotAuthorizedError occurs when the call is not authorized.
func NotAuthorizedError() error {
	return CommonError{
		error: errors.New("Not authorized"),
		Code:  http.StatusUnauthorized,
	}
}

// InvalidIDError occurs when an ID is not in a valid form.
func InvalidIDError() error {
	return CommonError{
		error: errors.New("ID is not in its proper form"),
		Code:  http.StatusBadRequest,
	}
}

// ValidationError occurs when there are internal validation errors, giving a message.
func ValidationError(msg string) error {
	return CommonError{
		error: errors.Errorf("Validation error: %s", msg),
		Code:  http.StatusBadRequest,
	}
}

// InvalidInputError occurs when external data validation fails, giving a suberror.
func InvalidInputError(err error) error {
	return CommonError{
		error: errors.Wrap(err, "Invalid input data"),
		Code:  http.StatusBadRequest,
	}
}

// EntityTooLargeError occurs when the input data is too large.
func EntityTooLargeError() error {
	return CommonError{
		error: errors.New("Request entity too large"),
		Code:  http.StatusRequestEntityTooLarge,
	}
}

func NotFoundError() error {
	return CommonError{
		error: errors.New("Entity not found"),
		Code:  http.StatusNotFound,
	}
}

// IsNotFoundError returns true if an error is an instance of a NotFoundError,
// or a wrapped instance of a NotFoundError.
func IsNotFoundError(err error) bool {
	err = errors.Cause(err)
	common, isCommon := err.(CommonError)
	return isCommon && common.Code == http.StatusNotFound
}

// Error handles all error responses for the API.
func Error(ctx context.Context, writer http.ResponseWriter, err error) {
	// Handle common errors with their specific messages
	if common, isCommon := err.(CommonError); isCommon {
		RespondError(ctx, writer, common.error, common.Code)
		return
	}

	// Handler server error by sending a general error to the client
	serverError := errors.Wrap(err, "an error has occurred. Try again")
	RespondError(ctx, writer, serverError, http.StatusInternalServerError)
}

// RespondError sends JSON describing the error
func RespondError(ctx context.Context, writer http.ResponseWriter, err error, code int) {
	// Log the error
	contextValues := ctx.Value(KeyValues).(*ContextValues)
	log.WithFields(log.Fields{
		"Method":     contextValues.Method,
		"RequestURI": contextValues.RequestURI,
		"TraceID":    contextValues.TraceID,
		"Code":       code,
		"Error":      fmt.Sprintf("%+v", err),
	}).Error("Server error")
	Respond(ctx, writer, JSONError{Error: err.Error()}, code)
}

// Respond sends JSON to the client.
// If code is StatusNoContent, v is expected to be nil.
func Respond(ctx context.Context, writer http.ResponseWriter, data interface{}, code int) {

	// Just set the status code and we are done.
	if code == http.StatusNoContent {
		writer.WriteHeader(code)
		return
	}
	if code == http.StatusCreated && data == nil {
		data = "Successful"
	}

	tracerID := ctx.Value(KeyValues).(*ContextValues).TraceID

	// Set the content type.
	writer.Header().Set("Content-Type", "application/json")

	// Write the status code to the response
	writer.WriteHeader(code)

	// Marshal the response data
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.WithFields(log.Fields{
			"Method":  "web.response",
			"Action":  "Marshal",
			"TraceId": tracerID,
			"Error":   err.Error(),
		}).Error("Error Marshalling JSON response")
		jsonData = []byte("{}")
	}

	// Send the result back to the client.
	_, err = writer.Write(jsonData)
	if err != nil {
		log.WithFields(log.Fields{
			"Method":  "web.response",
			"Action":  "Write",
			"TraceId": tracerID,
			"Error":   err.Error(),
		}).Error("Error writing response")
	}
}

// RespondHTML sends HTML to the client.
// If code is StatusNoContent, body is expected to be nil.
//
func RespondHTML(writer http.ResponseWriter, title string, body string, code int) {
	// Set the content type.
	writer.Header().Set("Content-Type", "text/html")

	// Write the status code to the response
	writer.WriteHeader(code)
	var buffer bytes.Buffer
	buffer.WriteString("<!DOCTYPE html><html><head><title>")
	buffer.WriteString(title)
	buffer.WriteString("</title></head><body>")
	buffer.WriteString(body)
	buffer.WriteString("</body></html>")

	_, err := writer.Write(buffer.Bytes())
	if err != nil {
		log.Printf("Failed to write the response body: %v", err)
		return
	}
}
