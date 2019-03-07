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

// Package client is an abstraction for connections to the bus
package client

import (
	"context_linux_go/core"
)

// ClientInterface represents an abstract connection to a broker server (bus) which distributes messages
// out to different clients
type ClientInterface interface { // nolint: golint
	EstablishConnection() error
	IsConnected() bool
	IsRegistered() bool
	Close()

	Publish(item *core.ItemData)
	RegisterDevice(onStartedHandler interface{})
	RegisterType(schema core.JSONSchema)
	SendCommand(macAddress string, application string, method string, params []interface{}, valueChannel chan interface{})
}

// CommandProcessor is an interface used by ClientInterface implementations to execute command handlers which have been
// registered with the Sensing instance
type CommandProcessor interface {
	ProcessCommand(contextType string, params []interface{}, returnChan chan interface{})
}

type credentials struct {
	NodeID   string `json:"node_id,omitempty"`
	Password string `json:"password,omitempty"`
}

type deviceInfo struct {
	MacAddress  string              `json:"MAC"`
	Sensors     *string             `json:"sensors"`
	Name        string              `json:"name"`
	Location    map[string]location `json:"location"`
	Credentials *credentials        `json:"credentials,omitempty"`
}

type serverParams struct {
	Handler string      `json:"handler"`
	Body    interface{} `json:"body"`
	Query   interface{} `json:"query"`
}

type result struct {
	Body         interface{} `json:"body"`
	ResponseCode int         `json:"response_code"`
}

type rpcError struct {
	Message interface{} `json:"message"`
	Code    int         `json:"code"`
}

// For REST-style messages (items)
type serverMessage struct {
	ContextRPC string        `json:"contextrpc"`
	JSONRPC    string        `json:"jsonrpc"`
	Method     *string       `json:"method,omitempty"`
	Endpoint   *endpoint     `json:"endpoint"`
	Params     *serverParams `json:"params"`
	ID         interface{}   `json:"id"`

	Result *result   `json:"result"`
	Error  *rpcError `json:"error"`
}

// For send/receive commands
type rpcMessage struct {
	ContextRPC string        `json:"contextrpc"`
	JSONRPC    string        `json:"jsonrpc"`
	Endpoint   *endpoint     `json:"endpoint"`
	Params     []interface{} `json:"params"`
	ID         interface{}   `json:"id"`
	Method     *string       `json:"method,omitempty"`

	Result *result   `json:"result"`
	Error  *rpcError `json:"error"`
}

type device struct {
	ID string `json:"id"`
}

type owner struct {
	Device device `json:"device"`
}

type state struct {
	DateTime string      `json:"dateTime"`
	Type     string      `json:"type"`
	Value    interface{} `json:"value"`
}

type serverRequestBody struct {
	Owner  owner   `json:"owner"`
	States []state `json:"states"`
}

type location struct {
	Country string `json:"country"`
	State   string `json:"state"`
	City    string `json:"city"`
}

type endpoint struct {
	MacAddress  string `json:"macaddress"`
	Application string `json:"application"`
}
