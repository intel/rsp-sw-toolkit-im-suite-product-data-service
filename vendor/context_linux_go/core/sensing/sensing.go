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

package sensing

import (
	"context_linux_go/core"
	"context_linux_go/core/client"
	"context_linux_go/logger"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"github.com/gorilla/websocket"
)

// Sensing integrates all the components of the SDK
type Sensing struct { // nolint: aligncheck

	// Local provider
	dispatchers map[int]*localDispatcher
	dispacherID int
	// Synchronize access to the two above fields
	dispMutex sync.Mutex

	// Remote channels
	itemChannels map[string][]*core.ProviderItemChannel
	errChannels  map[string][]*core.ErrorChannel
	// Synchronize add/remove access to the channel maps
	chanMutex sync.Mutex

	//Sensing Vars
	publish     bool
	application string
	macAddress  string
	server      string

	// Commands
	commandHandlers  map[int]core.CommandHandlerInfo
	commandHandlerID int
	nodeID           string
	password         string

	// Abstraction for talking to the clientInterface
	clientInterface client.ClientInterface
	clientItemChan  core.ProviderItemChannel
	clientErrChan   core.ErrorChannel

	remoteDone chan bool

	// Array of URNs which have had schemas registered with the clientInterface
	registeredCtxTypes map[string]bool

	// Event cache. String is either providerID:type or mac:application:type like AddContextTypeListener
	eventCache map[string]*core.ItemData
	useCache   bool //Based on options
	cacheMutex sync.Mutex

	//Sensing Channels
	startedC core.SensingStartedChannel
	errC     core.ErrorChannel

	logger logger.ContextGoLogger
}

// Represents a dispatcher for local providers
type localDispatcher struct {
	// Internal for the dispatcher goroutine
	internalItemChannel core.ProviderItemChannel
	internalErrChannel  core.ErrorChannel
	internalStopChannel chan bool

	// From the EnableSensing call. No filter on ContextType
	extItemChannel core.ProviderItemChannel
	extErrChannel  core.ErrorChannel

	// Reference to the provider object
	provider core.ProviderInterface
	// ID of this provider during the EnableSensing call
	providerID int
	// Reference to the sensing object which created this dispatcher
	sensing *Sensing
	// true iff sensing.Publish == true and provider.GetOptions().Publish == true
	publish bool
}

// NewSensing initializes a new instance of the Sensing core. Start() must be called before
// performing other operations
func NewSensing() *Sensing {
	sensing := Sensing{
		dispatchers:    make(map[int]*localDispatcher),
		itemChannels:   make(map[string][]*core.ProviderItemChannel),
		errChannels:    make(map[string][]*core.ErrorChannel),
		remoteDone:     make(chan bool),
		clientItemChan: make(core.ProviderItemChannel, 10),
		clientErrChan:  make(core.ErrorChannel, 10),

		registeredCtxTypes: make(map[string]bool),

		eventCache: make(map[string]*core.ItemData),

		logger: *logger.New("contextsdk", logger.JSONFormat, os.Stdout, logger.DebugLevel),

		commandHandlers: make(map[int]core.CommandHandlerInfo),
	}

	return &sensing
}

// Start begins local and remote sensing using the specified options
func (sensing *Sensing) Start(options core.SensingOptions) { // nolint: gocyclo

	if options.Application == "" {
		var err error
		options.Application, err = os.Hostname()
		if err != nil {
			//ToDo: Add error handling or logging
			println("Error during Sensing Start()")
		}
	}
	sensing.logger.EnableLogging()
	sensing.logger.SetLogLevel(options.LogLevel)
	if options.LogFile != "" {
		sensing.logger.SetLogFile(options.LogFile)
	}

	sensing.startedC = options.OnStarted
	sensing.errC = options.OnError

	sensing.server = options.Server

	sensing.clientInterface = client.NewWebSocketsClient(options, sensing.clientItemChan, sensing.clientErrChan, sensing)

	if sensing.clientInterface == nil {
		sensing.startedC <- &core.SensingStarted{Started: false}
		return
	}

	sensing.nodeID = options.NodeID
	sensing.password = options.Password

	sensing.publish = options.Publish

	sensing.useCache = options.UseCache

	builtIns := &builtinCommands{}
	sensing.EnableCommands(builtIns, map[string]interface{}{"publish": true, "sensing": sensing})

	sensingStarted := func() {
		started := core.SensingStarted{Started: true}
		sensing.startedC <- &started
		sensing.logger.Info("Sensing has started", nil)
	}

	// Acts as a mutex lock to prevent additional errors from attempting to establish connection
	attemptingReconnectAlready := false

	// Handle events coming in over the clientInterface item channel.
	// TODO distribute errors?
	go func() {
		for {
			select {
			case item := <-sensing.clientItemChan:
				specificKey := item.MacAddress + ":" + item.Application + ":" + item.Type
				genericKey := "*:*:" + item.Type
				itemChannels := sensing.itemChannels[specificKey]
				for _, itemChannel := range itemChannels {
					*itemChannel <- item
				}
				itemChannels = sensing.itemChannels[genericKey]
				for _, itemChannel := range itemChannels {
					*itemChannel <- item
				}

				if sensing.useCache {
					sensing.cacheMutex.Lock()
					sensing.eventCache[specificKey] = item
					sensing.eventCache[genericKey] = item
					sensing.cacheMutex.Unlock()
				}
			case <-sensing.remoteDone:
				return
			case wsErr := <-sensing.clientErrChan:
				if attemptingReconnectAlready {
					break
				}
				if websocket.IsUnexpectedCloseError(wsErr.Error, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					for {
						if sensing.clientInterface.IsConnected() {
							time.Sleep(time.Duration(options.RetryInterval * 1000) * time.Millisecond)
						} else {
							attemptingReconnectAlready = true
							attemptingReconnectAlready = sensing.establishConnection(nil)
							break
						}
					}
				} else if sensing.checkIfConnectionError(wsErr.Error.Error()){
					for {
						if sensing.clientInterface.IsConnected() {
							time.Sleep(time.Duration(options.RetryInterval * 1000) * time.Millisecond)
						} else {
							attemptingReconnectAlready = true
							attemptingReconnectAlready = sensing.establishConnection(nil)
							break
						}
					}
				} else {
					sensing.errC <- wsErr // Forward any other errors
				}
			}
		}
	}()

	if options.Server != "" {
		err := sensing.clientInterface.EstablishConnection()
		if err != nil {
			sensing.logger.Error(err.Error(), nil)
			sensing.errC <- core.ErrorData{Error: err}
		} else if !sensing.clientInterface.IsConnected() {
			sensing.logger.Panic("Websocket connection error. Program will exit.", logger.Params{"Broker Address": sensing.server})
			sensing.errC <- core.ErrorData{Error: fmt.Errorf(
				"Websocket connection error. Program will exit. Broker address: %v", sensing.server)}
		} else {
			sensing.clientInterface.RegisterDevice(sensingStarted)
		}
	} else {
		sensingStarted()
	}
}

// Stop local and remote sensing operations
func (sensing *Sensing) Stop() {
	sensing.logger.Info("Sensing is stopping", nil)
	sensing.clientInterface.Close()

	// Stop the remote dispatch goroutine
	sensing.remoteDone <- true

	// Stop Local providers
	for id := range sensing.dispatchers {
		err := sensing.DisableSensing(id)
		if err != nil {
			//ToDo: Add error handling or logging
			println("Error during Sensing Stop()")
		}
	}
}

// Establish WS connection with the provided options
func (sensing *Sensing) establishConnection(onStartHandler interface{}) bool{
	err := sensing.clientInterface.EstablishConnection()
	if err != nil {
		sensing.logger.Error(err.Error(), nil)
		sensing.errC <- core.ErrorData{Error: err}
	} else if !sensing.clientInterface.IsConnected() {
		sensing.logger.Panic("Websocket connection error. Program will exit.", logger.Params{"Broker Address": sensing.server})
		sensing.errC <- core.ErrorData{Error: fmt.Errorf(
			"Websocket connection error. Broker address: %v", sensing.server)}
	} else {
		sensing.clientInterface.RegisterDevice(onStartHandler)
	}
	return false
}

// Check if error message is related to connection failures
func (sensing *Sensing) checkIfConnectionError(errorMessage string) bool {
	return (strings.Contains(errorMessage, "read tcp") || (strings.Contains(errorMessage, "write tcp")))
}

// GetProviders returns a map of enabled local providers
//
// Key: Integer provider ID
//
// Value: Slice of URNs this provider supports
func (sensing *Sensing) GetProviders() map[int][]string {
	result := make(map[int][]string)

	sensing.dispMutex.Lock()
	defer sensing.dispMutex.Unlock()

	for _, disp := range sensing.dispatchers {
		var urns []string
		for _, t := range disp.provider.Types() {
			urns = append(urns, t.URN)
		}
		result[disp.providerID] = urns
	}

	return result
}

func (sensing *Sensing) isPublishing(providerID int) bool {
	sensing.dispMutex.Lock()
	defer sensing.dispMutex.Unlock()

	for _, disp := range sensing.dispatchers {
		if disp.providerID == providerID {
			return disp.publish
		}
	}

	return false
}

// Fill in the local provider item with the common fields before local distribution
// and remote publishing
func (sensing *Sensing) fillInItem(providerID int, item *core.ItemData) {
	if item != nil {
		item.ProviderID = providerID
		item.Application = sensing.application
		item.MacAddress = sensing.macAddress
		item.DateTime = time.Now().Format(time.RFC3339)
	}
}

// EnableSensing enables a specific local provider instance. All types this provider produces on its channel
// will be sent to onItem. Any errors will be sent to onErr. The channels may be nil.\
//
// Returns an integer representing the local provider ID, or -1 if the provider failed to start
func (sensing *Sensing) EnableSensing(provider core.ProviderInterface, onItem core.ProviderItemChannel, onErr core.ErrorChannel) int { // nolint: gocyclo
	sensing.dispMutex.Lock()
	defer sensing.dispMutex.Unlock()

	providerID := sensing.dispacherID + 1
	sensing.dispacherID++

	options := provider.GetOptions()
	options.ProviderID = providerID
	options.Sensing = sensing

	var dispatcher localDispatcher
	dispatcher.internalItemChannel = make(core.ProviderItemChannel, 5)
	dispatcher.internalErrChannel = make(core.ErrorChannel, 5)
	dispatcher.internalStopChannel = make(chan bool)
	dispatcher.extItemChannel = onItem
	dispatcher.extErrChannel = onErr
	dispatcher.provider = provider
	dispatcher.providerID = providerID
	dispatcher.sensing = sensing
	dispatcher.publish = options.Publish && sensing.publish

	sensing.dispatchers[providerID] = &dispatcher

	if dispatcher.publish {
		sensing.registerContextTypesWithBroker(provider)
	}

	// Must be buffered channels in case Start() writes item or err immediately
	// otherwise it will block and never return
	dispatcher.provider.Start(dispatcher.internalItemChannel, dispatcher.internalErrChannel)

	select {
	case err := <-dispatcher.internalErrChannel:
		outErr := fmt.Errorf("Provider failed to start: %s", err.Error.Error())
		sensing.logger.Error(outErr.Error(), nil)
		delete(sensing.dispatchers, providerID)
		sensing.dispacherID--
		sensing.errC <- core.ErrorData{Error: outErr}
		return -1
	default:
	}

	// Flags to see if the provider has closed its channels
	go func() {

		for {
			select {
			case item, ok := <-dispatcher.internalItemChannel:
				if ok {
					sensing.fillInItem(providerID, item)
					dispatcher.dispatch(providerID, item)
				} else {
					dispatcher.internalItemChannel = nil
				}
			case err, ok := <-dispatcher.internalErrChannel:
				if ok {
					dispatcher.dispatch(providerID, err)
				} else {
					dispatcher.internalErrChannel = nil
				}
			case <-dispatcher.internalStopChannel:
				return
			}

			if dispatcher.internalItemChannel == nil && dispatcher.internalErrChannel == nil {
				// Don't close the stop channel, if DisableSensing() is called
				// it will panic on the closed channel
				return
			}
		}
	}()

	return providerID
}

// DisableSensing disables a specific local provider which had been previously enabled with EnableSensing
func (sensing *Sensing) DisableSensing(providerID int) error {
	dispatcher, ok := sensing.dispatchers[providerID]

	// TODO Should send an error on the error channel instead?
	if !ok {
		sensing.logger.Error("Could not find provider.", logger.Params{"providerId": providerID})
		return fmt.Errorf("Could not find provider for %v", providerID)
	}

	dispatcher.provider.Stop()
	// Signal the goroutine to stop. Shouldn't need to close the channel since we
	// return in the dispatcher select() loop
	dispatcher.internalStopChannel <- true
	delete(sensing.dispatchers, providerID)

	return nil
}

// GetItem polls the provider for the latest item. It can optionally return the latest cached value (if any),
// and update the cache with the new result
func (sensing *Sensing) GetItem(providerID string, contextType string, useCache bool, updateCache bool) *core.ItemData {
	sensing.cacheMutex.Lock()
	defer sensing.cacheMutex.Unlock()

	if useCache {
		if !sensing.useCache {
			sensing.logger.Error("Cache is not enabled", nil)
			sensing.errC <- core.ErrorData{Error: errors.New("Cache not enabled")}
			return nil
		}

		return sensing.eventCache[providerID+":"+contextType]
	}

	if strings.Index(providerID, ":") > 0 {
		// Remote provider
		var remoteItem *core.ItemData // TODO get fresh value from remote
		if updateCache {
			sensing.eventCache[providerID+":"+contextType] = remoteItem
			sensing.eventCache["*.*:"+contextType] = remoteItem
		}
	} else {
		// Query Local provider
		pID, err := strconv.Atoi(providerID)
		if err == nil {
			dispatcher, ok := sensing.dispatchers[pID]
			if ok {
				item := dispatcher.provider.GetItem(contextType)
				sensing.fillInItem(pID, item)
				if updateCache {
					sensing.eventCache[providerID+":"+contextType] = item
				}
				return item
			}

			sensing.logger.Error("Local provider is not enabled", logger.Params{"Provider ID": pID})
			sensing.errC <- core.ErrorData{Error: fmt.Errorf("Local provider %d not enabled", pID)}
		} else {
			sensing.logger.Error("Could not parse provider ID", logger.Params{"error": err.Error()})
			sensing.errC <- core.ErrorData{Error: errors.New("Could not parse provider ID: " + err.Error())}
		}
	}

	return nil
}

// ProcessCommand is an internal implementation of the CommandProcessor interface. It should not be
// called directly
func (sensing *Sensing) ProcessCommand(contextType string, params []interface{}, valueChan chan interface{}) {
	handlerID := -1
	for hID := range sensing.commandHandlers {
		_, methodTypeFound := sensing.commandHandlers[hID].Methods[contextType]
		if sensing.commandHandlers[hID].Publish && methodTypeFound {
			handlerID = hID
			break
		}
	}

	if handlerID == -1 {
		valueChan <- core.ErrorData{Error: errors.New("No handlers found for " + contextType)}
		return
	}

	sensing.executeCommandHandler(handlerID, contextType, params, valueChan)
}

func (sensing *Sensing) executeCommandHandler(handlerID int, contextType string, params []interface{}, valueChan chan interface{}) {
	localHandlerInfo, ok := sensing.commandHandlers[handlerID]
	if !ok {
		valueChan <- core.ErrorData{Error: errors.New("Unknown Handler: " + strconv.Itoa(handlerID))}
	} else {
		localHandlerMethod := localHandlerInfo.Methods[contextType]

		// If the local command handler crashed, recover and report an error
		var returnedValue interface{}
		callDone := make(chan bool)
		go func() {
			defer func() {
				if err := recover(); err != nil {
					valueChan <- core.ErrorData{Error: fmt.Errorf("Command handler panicked! %v", err)}
					callDone <- false
					sensing.logger.StackTrace("Command handler panicked, recovered", err)
					return
				}
				callDone <- true
			}()

			returnedValue = localHandlerMethod(params...)
		}()
		success := <-callDone

		if success {
			valueChan <- returnedValue
		}
	}
}

// Dispatch a local provider's item to other local providers and optionally
// publish to the bus
// nolint: gocyclo
func (dispatcher *localDispatcher) dispatch(providerID int, data interface{}) {
	// Local Dispatch
	switch data.(type) {
	case *core.ItemData:
		// EnableSensing path
		item := data.(*core.ItemData)
		if dispatcher.extItemChannel != nil {
			dispatcher.extItemChannel <- item
		}

		// AddContextTypeListener path
		key := fmt.Sprintf("%v:%v", providerID, item.Type)
		for _, channel := range dispatcher.sensing.itemChannels[key] {
			*channel <- item
		}

		// Add item to event cache
		if dispatcher.sensing.useCache {
			dispatcher.sensing.cacheMutex.Lock()
			dispatcher.sensing.eventCache[key] = item
			dispatcher.sensing.cacheMutex.Unlock()
		}

		if dispatcher.publish && dispatcher.sensing.clientInterface.IsConnected() {
			dispatcher.sensing.clientInterface.Publish(item)
		}
	case core.ErrorData:
		err := data.(core.ErrorData)
		// EnableSensing path
		if dispatcher.extErrChannel != nil {
			dispatcher.extErrChannel <- err
		}

		// AddContextTypeListener path
		for _, channel := range dispatcher.sensing.errChannels[fmt.Sprintf("%v", providerID)] {
			*channel <- err
		}

		if dispatcher.publish && dispatcher.sensing.clientInterface.IsConnected() { // nolint: megacheck
			// TODO publish to clientInterface. Node version doesn't do this yet
		}
	default:
		dispatcher.sensing.logger.Panic("Unexpected type to dispatch.", logger.Params{"type": data})
		panic(fmt.Sprintf("Unexpected type to dispatch: %T", data))
	}
}

// Register the schema for each context type a provider supports with the clientInterface
// for schema validation
func (sensing *Sensing) registerContextTypesWithBroker(provider core.ProviderInterface) {
	if !sensing.clientInterface.IsConnected() {
		return
	}

	for _, provType := range provider.Types() {
		if _, ok := sensing.registeredCtxTypes[provType.URN]; !ok {
			sensing.clientInterface.RegisterType(provType.Schema)
			sensing.registeredCtxTypes[provType.URN] = true
		}
	}
}

// AddContextTypeListener adds a listener for a specific context type URN. Provider ID can be:
//
// macAddress:Application - for a specific remote provider
//
// *:* - for any remote provider
//
// Stringified integer provider ID - for local providers
func (sensing *Sensing) AddContextTypeListener(providerID string, contextType string, onItem *core.ProviderItemChannel, onErr *core.ErrorChannel) {
	eventKey := providerID + ":" + contextType

	sensing.chanMutex.Lock()
	defer sensing.chanMutex.Unlock()

	if onItem != nil {
		sensing.itemChannels[eventKey] = append(sensing.itemChannels[eventKey], onItem)
	}
	if onErr != nil {
		sensing.errChannels[providerID] = append(sensing.errChannels[providerID], onErr)
	}
}

// EnableCommands enables the command handled by the handler along with the options
func (sensing *Sensing) EnableCommands(handler core.CommandHandlerInterface, handlerStartOptions map[string]interface{}) int {
	handlerInfo := core.CommandHandlerInfo{
		Handler: handler,
		Methods: handler.Methods(),
		Publish: false,
	}
	if publish, ok := handlerStartOptions["publish"]; ok {
		handlerInfo.Publish = publish.(bool)
	}
	sensing.commandHandlers[sensing.commandHandlerID] = handlerInfo
	handler.Start(handlerStartOptions)
	returnID := sensing.commandHandlerID
	sensing.commandHandlerID++

	return returnID
}

// SendCommand redirects the parameters to the clientInterface's SendCommand function
func (sensing *Sensing) SendCommand(macAddress string, application string, handlerID int, method string, params []interface{}, valueChannel chan interface{}) {
	if macAddress == "" && application == "" {
		sensing.executeCommandHandler(handlerID, method, params, valueChannel)
	} else {
		sensing.clientInterface.SendCommand(macAddress, application, method, params, valueChannel)
	}
}