/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

package main

import (
	"time"

	_ "github.com/KimMachineGun/automemlimit" // By default, it sets `GOMEMLIMIT` to 90% of cgroup's memory limit.
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthealth"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/steadybit/extension-kit/extruntime"
	"github.com/steadybit/extension-kit/extsignals"
	"github.com/steadybit/extension-stackstate/config"
	"github.com/steadybit/extension-stackstate/extservice"
)

var startedAt = time.Now().Format(time.RFC3339)

func main() {
	extlogging.InitZeroLog()

	extbuild.PrintBuildInformation()
	extruntime.LogRuntimeInformation(zerolog.DebugLevel)

	exthealth.SetReady(false)
	exthealth.StartProbes(8084)

	config.ParseConfiguration()
	config.ValidateConfiguration()
	initStackStateHttpClient()

	discovery_kit_sdk.Register(extservice.NewServiceDiscovery())
	action_kit_sdk.RegisterAction(extservice.NewServiceStatusCheckAction())

	exthttp.RegisterHttpHandler("/", exthttp.IfNoneMatchHandler(func() string { return startedAt }, exthttp.GetterAsHandler(getExtensionList)))

	extsignals.ActivateSignalHandlers()

	action_kit_sdk.RegisterCoverageEndpoints()

	exthealth.SetReady(true)

	exthttp.Listen(exthttp.ListenOpts{
		Port: 8083,
	})
}

func initStackStateHttpClient() {
	client := resty.New()
	client.SetBaseURL(config.Config.ApiBaseUrl)
	client.SetHeader("X-API-Key", config.Config.ServiceToken)
	client.SetHeader("Content-Type", "application/json")
	extservice.Client = &extservice.StackStateHttpClient{
		Client: client,
	}
}

type ExtensionListResponse struct {
	action_kit_api.ActionList       `json:",inline"`
	discovery_kit_api.DiscoveryList `json:",inline"`
}

func getExtensionList() ExtensionListResponse {
	return ExtensionListResponse{
		ActionList:    action_kit_sdk.GetActionList(),
		DiscoveryList: discovery_kit_sdk.GetDiscoveryList(),
	}
}
