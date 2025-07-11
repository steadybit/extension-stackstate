/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package extservice

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-stackstate/config"
	"net/url"
	"strconv"
	"time"
)

type ServiceStatusCheckAction struct{}

// Make sure action implements all required interfaces
var (
	_ action_kit_sdk.Action[ServiceStatusCheckState]           = (*ServiceStatusCheckAction)(nil)
	_ action_kit_sdk.ActionWithStatus[ServiceStatusCheckState] = (*ServiceStatusCheckAction)(nil)
)

type ServiceStatusCheckState struct {
	ServiceId          string
	ServiceName        string
	ClusterName        string
	End                time.Time
	ExpectedStatus     string
	StatusCheckMode    string
	StatusCheckSuccess bool
}

type GetSnapshotApi interface {
	GetServiceSnapshot(ctx context.Context, serviceId string) (*resty.Response, ViewSnapshotResponseWrapper, error)
}

func NewServiceStatusCheckAction() action_kit_sdk.Action[ServiceStatusCheckState] {
	return &ServiceStatusCheckAction{}
}

func (m *ServiceStatusCheckAction) NewEmptyState() ServiceStatusCheckState {
	return ServiceStatusCheckState{}
}

func (m *ServiceStatusCheckAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          fmt.Sprintf("%s.check", serviceTargetType),
		Label:       "StackState Service",
		Description: "collects information about the service status and optionally verifies that the status has an expected status.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(serviceIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType:          serviceTargetType,
			QuantityRestriction: extutil.Ptr(action_kit_api.QuantityRestrictionAll),
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "service name",
					Description: extutil.Ptr("Find service by cluster, namespace and service"),
					Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.service.name=\"\"",
				},
			}),
		}),
		Technology:  extutil.Ptr("StackState"),
		Category:    extutil.Ptr("StackState"), //Can be removed in Q1/24 - support for backward compatibility of old sidebar
		Kind:        action_kit_api.Check,
		TimeControl: action_kit_api.TimeControlInternal,
		Parameters: []action_kit_api.ActionParameter{
			{
				Name:         "duration",
				Label:        "Duration",
				Description:  extutil.Ptr(""),
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: extutil.Ptr("30s"),
				Order:        extutil.Ptr(1),
				Required:     extutil.Ptr(true),
			},
			{
				Name:        "expectedStatus",
				Label:       "Expected Status",
				Description: extutil.Ptr(""),
				Type:        action_kit_api.ActionParameterTypeString,
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "CLEAR",
						Value: "CLEAR",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "DEVIATING",
						Value: "DEVIATING",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "CRITICAL",
						Value: "CRITICAL",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "UNKNOWN",
						Value: "UNKNOWN",
					},
				}),
				Required: extutil.Ptr(false),
				Order:    extutil.Ptr(2),
			},
			{
				Name:         "statusCheckMode",
				Label:        "Status Check Mode",
				Description:  extutil.Ptr("How often should the status be expected?"),
				Type:         action_kit_api.ActionParameterTypeString,
				DefaultValue: extutil.Ptr(statusCheckModeAllTheTime),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "All the time",
						Value: statusCheckModeAllTheTime,
					},
					action_kit_api.ExplicitParameterOption{
						Label: "At least once",
						Value: statusCheckModeAtLeastOnce,
					},
				}),
				Required: extutil.Ptr(true),
				Order:    extutil.Ptr(4),
			},
		},
		Widgets: extutil.Ptr([]action_kit_api.Widget{
			action_kit_api.StateOverTimeWidget{
				Type:  action_kit_api.ComSteadybitWidgetStateOverTime,
				Title: "StackState Service Status",
				Identity: action_kit_api.StateOverTimeWidgetIdentityConfig{
					From: attributeServiceId,
				},
				Label: action_kit_api.StateOverTimeWidgetLabelConfig{
					From: attributeK8ServiceName,
				},
				State: action_kit_api.StateOverTimeWidgetStateConfig{
					From: attributeState,
				},
				Tooltip: action_kit_api.StateOverTimeWidgetTooltipConfig{
					From: attributeTooltip,
				},
				Url: extutil.Ptr(action_kit_api.StateOverTimeWidgetUrlConfig{
					From: extutil.Ptr(attributeUrl),
				}),
				Value: extutil.Ptr(action_kit_api.StateOverTimeWidgetValueConfig{
					Hide: extutil.Ptr(true),
				}),
			},
		}),
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("1s"),
		}),
	}
}

func (m *ServiceStatusCheckAction) Prepare(_ context.Context, state *ServiceStatusCheckState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	serviceId := request.Target.Attributes[attributeServiceId]
	if len(serviceId) == 0 {
		return nil, extutil.Ptr(extension_kit.ToError("Target is missing the 'stackstate.service.id' attribute.", nil))
	}

	duration := request.Config["duration"].(float64)
	end := time.Now().Add(time.Millisecond * time.Duration(duration))

	var expectedStatus string
	if request.Config["expectedStatus"] != nil {
		expectedStatus = fmt.Sprintf("%v", request.Config["expectedStatus"])
	}
	var statusCheckMode = statusCheckModeAllTheTime
	if request.Config["statusCheckMode"] != nil {
		statusCheckMode = fmt.Sprintf("%v", request.Config["statusCheckMode"])
	}

	state.ServiceId = serviceId[0]
	state.ServiceName = request.Target.Attributes[attributeK8ServiceName][0]
	state.ClusterName = request.Target.Attributes[attributeK8ClusterName][0]
	state.End = end
	state.ExpectedStatus = expectedStatus
	state.StatusCheckMode = statusCheckMode
	state.StatusCheckSuccess = state.StatusCheckMode == statusCheckModeAllTheTime

	return nil, nil
}

func (m *ServiceStatusCheckAction) Start(_ context.Context, _ *ServiceStatusCheckState) (*action_kit_api.StartResult, error) {
	return nil, nil
}

func (m *ServiceStatusCheckAction) Status(ctx context.Context, state *ServiceStatusCheckState) (*action_kit_api.StatusResult, error) {
	return MonitorStatusCheckStatus(ctx, state, Client)
}

func MonitorStatusCheckStatus(ctx context.Context, state *ServiceStatusCheckState, api GetSnapshotApi) (*action_kit_api.StatusResult, error) {
	now := time.Now()
	component, err := loadServiceComponent(ctx, state, api)
	if err != nil {
		return nil, err
	}
	completed := now.After(state.End)

	var checkError *action_kit_api.ActionKitError
	if len(state.ExpectedStatus) > 0 {
		componentHealthState := component.State.HealthState
		if state.StatusCheckMode == statusCheckModeAllTheTime && componentHealthState != state.ExpectedStatus {
			checkError = extutil.Ptr(action_kit_api.ActionKitError{
				Title: fmt.Sprintf("Service '%s' (id %s) has status '%s' whereas '%s' is expected.",
					component.Name,
					state.ServiceId,
					componentHealthState,
					state.ExpectedStatus),
				Status: extutil.Ptr(action_kit_api.Failed),
			})
		} else if state.StatusCheckMode == statusCheckModeAtLeastOnce {
			if componentHealthState == state.ExpectedStatus {
				state.StatusCheckSuccess = true
			}
			if completed && !state.StatusCheckSuccess {
				checkError = extutil.Ptr(action_kit_api.ActionKitError{
					Title: fmt.Sprintf("Service '%s' (id %s) didn't have status '%s' at least once.",
						component.Name,
						state.ServiceId,
						state.ExpectedStatus),
					Status: extutil.Ptr(action_kit_api.Failed),
				})
			}
		}
	}

	return &action_kit_api.StatusResult{
		Completed: completed,
		Error:     checkError,
		Metrics: &[]action_kit_api.Metric{
			*toMetric(component, now),
		},
	}, nil
}

func loadServiceComponent(ctx context.Context, state *ServiceStatusCheckState, api GetSnapshotApi) (*Component, error) {
	res, stackStateResponse, err := api.GetServiceSnapshot(ctx, state.ServiceId)
	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError(fmt.Sprintf("Failed to retrieve service states from StackState for Service ID %s. Full response: %v", state.ServiceId, res.String()), err))
	}
	if !res.IsSuccess() {
		log.Err(err).Msgf("StackState API responded with unexpected status code %d while retrieving service states for Service ID %s. Full response: %v", res.StatusCode(), state.ServiceId, res.String())
		serviceIdInt, parseErr := strconv.Atoi(state.ServiceId)
		if parseErr != nil {
			return nil, extutil.Ptr(extension_kit.ToError(fmt.Sprintf("Failed to parse int %s", state.ServiceId), parseErr))
		}
		return &Component{
			Id:   serviceIdInt,
			Name: state.ServiceName,
			State: State{
				HealthState: "UNKNOWN",
			},
			Identifiers: []string{fmt.Sprintf("urn:service:/%s:%s:%s", state.ClusterName, state.ServiceName, state.ServiceId)},
		}, nil
	}
	return &stackStateResponse.ViewSnapshotResponse.Components[0], nil
}

func toMetric(service *Component, now time.Time) *action_kit_api.Metric {
	var tooltip string
	var state string

	tooltip = fmt.Sprintf("Service status is: %s", service.State.HealthState)
	if service.State.HealthState == "UNKNOWN" {
		state = "warn"
	} else if service.State.HealthState == "CLEAR" {
		state = "success"
	} else if service.State.HealthState == "DEVIATING" {
		state = "warn"
	} else if service.State.HealthState == "CRITICAL" {
		state = "danger"
	}

	uiBaseUrl := config.Config.ApiBaseUrl[:(len(config.Config.ApiBaseUrl) - 3)]

	return extutil.Ptr(action_kit_api.Metric{
		Name: extutil.Ptr("stackstate_service_status"),
		Metric: map[string]string{
			attributeServiceId:     strconv.Itoa(service.Id),
			attributeK8ServiceName: service.Name,
			attributeState:         state,
			attributeTooltip:       tooltip,
			attributeUrl:           fmt.Sprintf("%s/#/components/%s", uiBaseUrl, url.QueryEscape(service.Identifiers[0])),
		},
		Timestamp: now,
		Value:     0,
	})
}
