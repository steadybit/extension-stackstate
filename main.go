/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package main

import (
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthealth"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/steadybit/extension-scaffold/config"
)

func main() {
	extlogging.InitZeroLog()

	extbuild.PrintBuildInformation()

	exthealth.SetReady(false)
	exthealth.StartProbes(8084)

	config.ParseConfiguration()
	config.ValidateConfiguration()

	exthttp.RegisterHttpHandler("/", exthttp.GetterAsHandler(getExtensionList))

	//extrobots.RegisterDiscoveryHandlers()
	//action_kit_sdk.RegisterAction(extrobots.NewLogAction())

	action_kit_sdk.InstallSignalHandler()

	action_kit_sdk.RegisterCoverageEndpoints()

	exthealth.SetReady(true)

	exthttp.Listen(exthttp.ListenOpts{
		Port: 8083,
	})
}

type ExtensionListResponse struct {
	action_kit_api.ActionList       `json:",inline"`
	discovery_kit_api.DiscoveryList `json:",inline"`
}

func getExtensionList() ExtensionListResponse {
	return ExtensionListResponse{
		ActionList: action_kit_sdk.GetActionList(),
		//DiscoveryList: extrobots.GetDiscoveryList(),
	}
}
