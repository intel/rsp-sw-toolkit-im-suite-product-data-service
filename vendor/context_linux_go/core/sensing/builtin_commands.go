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
	"fmt"
)

const (
	// GetItemCommand is the URI for a command that requests the remote Sensing instance to return a new item
	// from a provider.
	// Arguments: Identical to core.Sensing.GetItem (including the provider ID, which may be any empty string)
	// Return value: A map[string]interface{} with keys matching fields in core.ItemData
	GetItemCommand string = "urn:x-intel:context:command:getitem"

	// FindResourcesCommand is the URI for a command that returns the context types published by the Sensing instance.
	// Arguments: None
	// Return value: Array of context type strings
	FindResourcesCommand string = "urn:x-intel:context:command:findresources"
)

type builtinCommands struct {
	sensing *Sensing
}

// Start function
func (bic *builtinCommands) Start(options map[string]interface{}) {
	bic.sensing = options["sensing"].(*Sensing)
}

// Stop function
func (bic *builtinCommands) Stop() {

}

// Methods function
func (bic *builtinCommands) Methods() map[string]core.CommandFunc {
	methodMap := make(map[string]core.CommandFunc)
	methodMap[GetItemCommand] = func(params ...interface{}) interface{} {
		return bic.handleGetItemCommand(params)
	}
	methodMap[FindResourcesCommand] = func(params ...interface{}) interface{} {
		return bic.handleFindResourcesCommand(params)
	}
	return methodMap
}

func (bic *builtinCommands) handleGetItemCommand(params []interface{}) interface{} {
	if len(params) < 2 || len(params) > 4 {
		return core.ErrorData{Error: fmt.Errorf("Requires 2-4 arguments, received %d", len(params))}
	}

	contextType, ok := params[1].(string)

	for providerID, providerURNs := range bic.sensing.GetProviders() {
		if ok {
			for _, urn := range providerURNs {
				if urn == contextType {
					var useCache, updateCache bool

					if len(params) > 2 {
						useCache, _ = params[2].(bool)
						updateCache, _ = params[3].(bool)
					}

					item := bic.sensing.GetItem(fmt.Sprint(providerID), contextType, useCache, updateCache)
					return *item
				}
			}
		}
	}

	return core.ErrorData{Error: fmt.Errorf("No data found for context type %s", contextType)}
}

func (bic *builtinCommands) handleFindResourcesCommand(params []interface{}) interface{} {
	providers := bic.sensing.GetProviders()
	results := []string{}
	seen := map[string]bool{}
	for providerID, provTypes := range providers {
		for _, uri := range provTypes {
			if _, ok := seen[uri]; !ok {
				if bic.sensing.isPublishing(providerID) {
					results = append(results, uri)
					seen[uri] = true
				}
			}
		}
	}

	return results
}