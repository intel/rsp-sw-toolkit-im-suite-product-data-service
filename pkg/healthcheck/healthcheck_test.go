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
package healthcheck

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

var status = "healthy"

func TestHealthcheck_Healthy(t *testing.T) {
	status = "healthy"
	client := http.DefaultClient
	client.Transport = newMockTransport()
	status := Healthcheck("80")
	if status == 1 {
		t.Error("Healthcheck healthy status should return 0")
	}
}
func TestHealthcheck_Unhealthy(t *testing.T) {
	status = "unhealthy"
	client := http.DefaultClient
	client.Transport = newMockTransport()
	status := Healthcheck("80")
	if status == 0 {
		t.Error("Healthcheck unhealthy status should return 1")
	}

}

type mockTransport struct{}

func newMockTransport() http.RoundTripper {
	return &mockTransport{}
}

// Implement http.RoundTripper
func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	statusCode := 200
	if status == "healthy" {
		statusCode = 200 // http.StatusOK
	} else if status == "unhealthy" {
		statusCode = 500
	}
	// Create mocked http.Response
	response := &http.Response{
		Header:     make(http.Header),
		Request:    req,
		StatusCode: statusCode,
	}
	response.Header.Set("Content-Type", "application/json")
	response.Body = ioutil.NopCloser(strings.NewReader("Service running"))
	return response, nil
}
