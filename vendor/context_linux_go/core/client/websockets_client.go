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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	logger "context_linux_go/logger"
	"github.com/gorilla/websocket"
	"net/http"
)

var contextRPCVersion = core.GetVersion()

const (
	jsonRPCVersion     = "2.0"
	defaultIPAddress   = "0:0:0:0:0:0"
	defaultApplication = "sensing"
)

// Global so it can be overridden by
var wsDialer = func(dialer websocket.Dialer, serverURL string, requestHeader http.Header) (*websocket.Conn, *http.Response, error) {
	return dialer.Dial(serverURL, requestHeader)
}

// WSClient represents a websocket wsClient connection instance
type WSClient struct { // nolint: aligncheck
	itemChan         core.ProviderItemChannel
	errChan          core.ErrorChannel
	application      string
	macAddress       string
	server           string
	connectionPrefix string
	requestID        int
	serverConnection *websocket.Conn
	keepAlive        bool
	doneChan         chan bool
	jsonMessageChan  chan interface{}
	pingChan         chan string

	isConnected        bool
	isDeviceRegistered bool
	responseHandlers   map[int]interface{}

	nodeID   string
	password string

	rootCA           *x509.CertPool
	skipVerification bool

	retries       int
	retryInterval int
	localogger    logger.ContextGoLogger

	respHandlerMutex sync.Mutex
	connCheckMutex   sync.Mutex

	cmdProc  CommandProcessor
	pongWait time.Duration
}

// NewWebSocketsClient creates a new wsClient connection using websockets
func NewWebSocketsClient(options core.SensingOptions, onItem core.ProviderItemChannel, onErr core.ErrorChannel, cmdProc CommandProcessor) ClientInterface {
	var wsClient WSClient

	if options.Secure {
		wsClient.connectionPrefix = "wss"

		if options.RootCertificates != nil {
			wsClient.rootCA = x509.NewCertPool()
			for _, cert := range options.RootCertificates {
				wsClient.rootCA.AddCert(cert)
			}
		}

		wsClient.skipVerification = options.SkipCertificateVerification
	} else {
		wsClient.connectionPrefix = "ws"
	}

	wsClient.application = options.Application
	wsClient.macAddress = options.Macaddress
	wsClient.server = options.Server

	wsClient.server = options.Server
	wsClient.itemChan = onItem
	wsClient.errChan = onErr
	wsClient.retries = options.Retries
	wsClient.retryInterval = options.RetryInterval

	wsClient.responseHandlers = make(map[int]interface{})

	wsClient.nodeID = options.NodeID
	wsClient.password = options.Password

	// If the connection loop has an error, this channel will be written to before Close()
	wsClient.doneChan = make(chan bool)

	wsClient.cmdProc = cmdProc

	wsClient.jsonMessageChan = make(chan interface{}, 100000)
	wsClient.pingChan = make(chan string, 1)
	wsClient.localogger = *logger.New("saf-websocket client", logger.JSONFormat, os.Stdout, logger.DebugLevel)
	wsClient.pongWait = 60 * time.Second
	go func() {
		for {
			// below code is to make sure ping handler gets preference over writeJson
			select {
			case <-wsClient.pingChan:
				wsClient.writePong()
			default:
			}
			select {
			case <-wsClient.pingChan:
				wsClient.writePong()
			case jsonStr := <-wsClient.jsonMessageChan:
				if wsClient.serverConnection != nil {
					wsClient.localogger.Info("sending message to broker", logger.Params{"lengthOfChanBuffer": len(wsClient.jsonMessageChan)})
					err := wsClient.serverConnection.WriteJSON(jsonStr)
					if err != nil {
						wsClient.handleKeepAliveIfErrorDuringReadOrWrite(err)
					}
				}
			}
		}
	}()

	return &wsClient
}

func (wsClient *WSClient) writePong() {
	if wsClient.serverConnection != nil {
		wsClient.localogger.Info("Responding with Pong", nil)
		wsClient.serverConnection.WriteMessage(websocket.PongMessage, []byte{})
	}
}

// Publish sends a new item to the wsClient to be distributed
func (wsClient *WSClient) Publish(item *core.ItemData) {
	pubRequest := serverRequestBody{
		Owner: owner{
			Device: device{
				ID: wsClient.macAddress + ":" + wsClient.application,
			},
		},
		States: []state{
			state{
				Type:     item.Type,
				DateTime: item.DateTime,
				Value:    item.Value,
			},
		},
	}

	wsClient.sendToServer("PUT", "states", "", pubRequest, nil)
}

// EstablishConnection tries to connect to the wsClient and returns an error if it is
// unsuccessful
func (wsClient *WSClient) EstablishConnection() error {
	serverURL := fmt.Sprintf("%s://%s/context/v1/socket", wsClient.connectionPrefix, wsClient.server)

	wsHeaders := http.Header{
		"Origin": {"http://localhost/"},
	}

	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			RootCAs:            wsClient.rootCA,
			InsecureSkipVerify: wsClient.skipVerification, // nolint: gas
		},
	}

	var returnError error

	for i := 1; i <= wsClient.retries+1; i++ {
		connection, _, err := wsDialer(dialer, serverURL, wsHeaders)
		if err != nil || connection == nil {
			fmt.Printf("Websocket connection error. Attempt: %v / %v\n", i, wsClient.retries+1)
			if err != nil {
				fmt.Println(err)
			}
			time.Sleep(time.Duration(wsClient.retryInterval) * time.Second)
			returnError = err
		} else {
			returnError = nil
			wsClient.connCheckMutex.Lock()
			wsClient.serverConnection = connection
			wsClient.isConnected = true
			wsClient.connCheckMutex.Unlock()
			break
		}
	}
	return returnError
}

// IsConnected returns true if there is an active connection to the wsClient
func (wsClient *WSClient) IsConnected() bool {
	wsClient.connCheckMutex.Lock()
	defer wsClient.connCheckMutex.Unlock()
	return wsClient.isConnected
}

// IsRegistered returns true if there is an active device registration to the broker
func (wsClient *WSClient) IsRegistered() bool {
	return wsClient.isDeviceRegistered
}

// Close ends the connection to the wsClient
func (wsClient *WSClient) Close() {
	wsClient.keepAlive = false

	if wsClient.serverConnection != nil {
		err := wsClient.serverConnection.Close()

		if err != nil {
			wsClient.errChan <- core.ErrorData{Error: errors.New("Error closing connection")}
		}
		wsClient.serverConnection = nil

		// Wait until the listener goroutine is done
		<-wsClient.doneChan
	}
}

func (wsClient *WSClient) sendCommandRPC(method string, params []interface{}, responseHandler interface{}, macAddress string, application string) {

	if macAddress == "" {
		macAddress = defaultIPAddress
	}
	if application == "" {
		application = defaultApplication
	}

	if responseHandler != nil {
		wsClient.responseHandlers[wsClient.requestID] = responseHandler
	}

	requestID := wsClient.requestID

	jsonStr := rpcMessage{
		ContextRPC: contextRPCVersion,
		JSONRPC:    jsonRPCVersion,
		Method:     &method,
		Endpoint: &endpoint{
			MacAddress:  macAddress,
			Application: application,
		},
		Params: params,
		ID:     &requestID,
	}
	//wsClient.localogger.Debug("sendCommandRPC", logger.Params{"requestID": wsClient.requestID})
	wsClient.requestID++
	wsClient.jsonMessageChan <- jsonStr
}

func (wsClient *WSClient) sendCommand(method string, params serverParams, responseHandler interface{}, macAddress string, application string) {

	if macAddress == "" {
		macAddress = defaultIPAddress
	}
	if application == "" {
		application = defaultApplication
	}

	if responseHandler != nil {
		wsClient.responseHandlers[wsClient.requestID] = responseHandler
	}

	requestID := wsClient.requestID

	jsonStr := serverMessage{
		ContextRPC: contextRPCVersion,
		JSONRPC:    jsonRPCVersion,
		Method:     &method,
		Endpoint: &endpoint{
			MacAddress:  macAddress,
			Application: application,
		},
		Params: &params,
		ID:     &requestID,
	}
	wsClient.jsonMessageChan <- jsonStr
	//wsClient.localogger.Debug("sendCommand", logger.Params{"requestID": wsClient.requestID, "lengthOfChanBuffer": len(wsClient.jsonMessageChan)})
	wsClient.requestID++
}

func (wsClient *WSClient) sendToServer(method string, handler string, query string, body interface{}, responseHandler interface{}) {

	params := serverParams{
		Handler: handler,
		Body:    body,
	}

	if query == "" {
		params.Query = nil
	} else {
		params.Query = &query
	}

	wsClient.sendCommand(method, params, responseHandler, "", "")
}

func (wsClient *WSClient) webSocketListener() { //nolint: gocyclo

	wsClient.keepAlive = true

	//Ping Pong Handler
	wsClient.serverConnection.SetPingHandler(
		func(m string) error {
			wsClient.localogger.Info("ping recieved", nil)
			wsClient.pingChan <- m
			return nil
		})

	for wsClient.keepAlive {

		if wsClient.serverConnection == nil {
			wsClient.errChan <- core.ErrorData{Error: errors.New("Websocket Error - Connection Does Not Exist")}
			break
		}
		_, msg, err := wsClient.serverConnection.ReadMessage()
		if err != nil {
			// If keepAlive is false, we are in the shutdown state and so can recieve errors relating
			// to a closed connection which should be ignored (since there isn't another way to break out of Receive)
			wsClient.handleKeepAliveIfErrorDuringReadOrWrite(err)

			// Break on error always since message is not valid
			break
		}
		rawMessage := map[string]*json.RawMessage{}
		err = json.Unmarshal(msg, &rawMessage)

		// Unmarshal the rest of the message based on its keys (request, response, method)
		if err == nil {
			if isResponse(&rawMessage) {
				srvMessage := serverMessage{}
				err = json.Unmarshal(msg, &srvMessage)
				if srvMessage.Error != nil {
					wsClient.localogger.Debug(srvMessage.Error.Message.(string), nil)
					wsClient.errChan <- core.ErrorData{Error: fmt.Errorf("Received error response: %s %d",
						srvMessage.Error.Message, srvMessage.Error.Code)}
				}
				wsClient.handleResponse(&srvMessage)
			} else {
				// See if this request is an RPC request
				method := ""
				err = json.Unmarshal(*rawMessage["method"], &method)
				if err == nil {
					if strings.HasPrefix(method, "RPC:") {
						rpcMsg := rpcMessage{}
						err = json.Unmarshal(msg, &rpcMsg)
						wsClient.handleRPCRequest(&rpcMsg)
					} else {
						// Item PUT broadcast from the broker
						srvMessage := serverMessage{}
						err = json.Unmarshal(msg, &srvMessage)
						wsClient.handleItemPut(&srvMessage)
					}
				}
			}
		}

		if err != nil {
			wsClient.errChan <- core.ErrorData{Error: fmt.Errorf("Could not parse message: %s", err)}
		}
	}

	wsClient.doneChan <- true
}

func (wsClient *WSClient) handleItemPut(message *serverMessage) { // nolint: gocyclo
	if (message.Endpoint.MacAddress == defaultIPAddress) && (message.Endpoint.Application == defaultApplication) && (*message.Method == "PUT") && (message.Params.Handler == "states") {
		body := serverRequestBody{}

		bytes, err := json.Marshal(message.Params.Body)
		if err != nil {
			wsClient.errChan <- core.ErrorData{Error: errors.New("Error on JSON marshall")}
		}
		err = json.Unmarshal(bytes, &body)
		if err != nil {
			wsClient.errChan <- core.ErrorData{Error: errors.New("Error on JSON unmarshall")}
		}

		macaddress, application := splitDeviceID(body.Owner.Device.ID)

		for _, stateItem := range body.States {
			itemData := core.ItemData{
				MacAddress:  macaddress,
				Application: application,
				ProviderID:  -1,
				DateTime:    stateItem.DateTime,
				Type:        stateItem.Type,
				Value:       stateItem.Value,
			}

			wsClient.itemChan <- &itemData
		}
	}
}

func (wsClient *WSClient) handleResponse(message *serverMessage) {

	var responseID int
	var responseIDFloat float64
	var responseIDString string
	var splitResponseID []string
	var err error
	var ok bool
	var msg = *message
	var messageID interface{} = msg.ID
	responseIDFloat, ok = messageID.(float64)
	if !ok {
		responseIDString, ok = messageID.(string)
		if !ok {
			panic("message ID was not a string or int from wsClient")
		}
		splitResponseID = strings.Split(responseIDString, ":")
		responseID, err = strconv.Atoi(splitResponseID[len(splitResponseID)-1])
		if err != nil {
			panic("could not convert message ID to int")
		}
	} else {
		responseID = int(responseIDFloat)
	}
	responseHandler := wsClient.responseHandlers[responseID]
	if responseHandler != nil {
		handler := reflect.ValueOf(responseHandler)
		var handlerArgs []reflect.Value
		handlerArgs = append(handlerArgs, reflect.ValueOf(*message))
		handler.Call(handlerArgs)
	}
	delete(wsClient.responseHandlers, responseID)
}

// SendCommand sends the value to the local or remote handler
func (wsClient *WSClient) SendCommand(macAddress string, application string, method string, params []interface{}, valueChannel chan interface{}) { //nolint: gocyclo
	if macAddress == "" && application == "" {
		panic("Local commands not implemented by the client interface")
	} else {
		//remote handler
		sendCommandResponseHandler := func(response serverMessage) {
			if response.Error != nil {
				err := fmt.Errorf("SendCommand Error: %v", response.Error)
				valueChannel <- core.ErrorData{Error: err}
				return
			}

			valueChannel <- response.Result.Body
		}
		wsClient.sendCommandRPC("RPC:"+method, params, sendCommandResponseHandler, macAddress, application)
	}
}

func (wsClient *WSClient) handleRPCRequest(request *rpcMessage) {
	methodType := strings.Trim(*request.Method, "RPC:")

	arrayParams := request.Params
	valueChannel := make(chan interface{}, 1)

	wsClient.cmdProc.ProcessCommand(methodType, arrayParams, valueChannel)

	genericValue := <-valueChannel
	//is local error?
	errorValue, ok := genericValue.(core.ErrorData)
	if ok {
		wsClient.sendRPCResponse(request, nil, &rpcError{Message: errorValue.Error.Error(), Code: http.StatusBadRequest})
		return
	}

	wsClient.sendRPCResponse(request, &result{Body: genericValue, ResponseCode: http.StatusOK}, nil)
}

func (wsClient *WSClient) sendRPCResponse(request *rpcMessage, result *result, errorResult *rpcError) {
	response := rpcMessage{
		ContextRPC: contextRPCVersion,
		ID:         request.ID,
		JSONRPC:    jsonRPCVersion,
	}
	if errorResult != nil {
		response.Error = errorResult
	} else {
		response.Result = result
	}
	//wsClient.localogger.Debug("sendRPCResponse", nil)
	wsClient.jsonMessageChan <- response
}

func isResponse(message *map[string]*json.RawMessage) bool {
	return (*message)["result"] != nil || (*message)["error"] != nil
}

func (wsClient *WSClient) handleKeepAliveIfErrorDuringReadOrWrite(err error) {
	if wsClient.keepAlive {
		wsClient.errChan <- core.ErrorData{Error: err}
		if wsClient.serverConnection != nil {
			err := wsClient.serverConnection.Close() //nolint:vetshadow
			if err != nil {
				wsClient.errChan <- core.ErrorData{Error: errors.New("Error on Server Connection Close")}
			}
			wsClient.connCheckMutex.Lock()
			wsClient.isConnected = false
			wsClient.isDeviceRegistered = false
			wsClient.serverConnection = nil
			wsClient.connCheckMutex.Unlock()
		}
	}
}
