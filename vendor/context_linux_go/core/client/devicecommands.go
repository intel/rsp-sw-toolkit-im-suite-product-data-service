//
// INTEL CONFIDENTIAL
// Copyright 2017 Intel Corporation.
// 
// This software and the related documents are Intel copyrighted materials, and your use of them is governed
// by the express license under which they were provided to you (License). Unless the License provides otherwise,
// you may not use, modify, copy, publish, distribute, disclose or transmit this software or the related documents
// without Intel's prior written permission.
// 
// This software and the related documents are provided as is, with no express or implied warranties, other than
// those that are expressly stated in the License.
//

package client

import (
	"context_linux_go/core"
	"errors"
	"net"
	"reflect"
	"strings"
)

// RegisterDevice registers this application with the wsClient, and calls onStartedHandler
// once the connection has been established
func (wsClient *WSClient) RegisterDevice(onStartedHandler interface{}) {

	clientDeviceInfo := wsClient.getDeviceInfo()
	wsClient.macAddress = clientDeviceInfo.MacAddress

	go wsClient.webSocketListener()

	registrationResponseHandler := func(response serverMessage) {
		if response.Error != nil {
			wsClient.errChan <- core.ErrorData{Error: errors.New("Registration error")}
		}

		if onStartedHandler != nil {
			handler := reflect.ValueOf(onStartedHandler)
			handler.Call(nil)
		}

		wsClient.isDeviceRegistered = true
	}

	wsClient.sendToServer("PUT", "devices", "", clientDeviceInfo, registrationResponseHandler)

}

func (wsClient *WSClient) getDeviceInfo() deviceInfo {
	macAddress := ""
	interfaces, _ := net.Interfaces()
	for _, item := range interfaces {
		if len(item.HardwareAddr) > 0 {
			macAddress = item.HardwareAddr.String()
			//TODO: Validate MAC Address
			break
		}
	}

	if macAddress == "" {
		wsClient.errChan <- core.ErrorData{Error: errors.New("MAC Address is Empty")}
	}

	loc := make(map[string]location)
	loc["semantic"] = location{
		Country: "US",
		State:   "OR",
		City:    "Hillsboro",
	}

	// If credentials are not provided, this will still be nil. When marshalling,
	// the field will be ignored since it is a pointer
	var creds *credentials

	if wsClient.nodeID != "" && wsClient.password != "" {
		creds = &credentials{
			NodeID:   wsClient.nodeID,
			Password: wsClient.password,
		}
	}

	devInfo := deviceInfo{
		MacAddress:  macAddress,
		Sensors:     nil,
		Name:        wsClient.application,
		Location:    loc,
		Credentials: creds,
	}

	return devInfo
}

func splitDeviceID(deviceID string) (string, string) {
	macaddress := strings.Join(strings.Split(deviceID, ":")[:6], ":")
	application := strings.Split(deviceID, ":")[6]

	return macaddress, application
}

// RegisterType registers a provider schema with the wsClient to perform validation
func (wsClient *WSClient) RegisterType(schema core.JSONSchema) {
	wsClient.sendToServer("PUT", "typescatalog", "", schema, nil)
}