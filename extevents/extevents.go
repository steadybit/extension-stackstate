// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extevents

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/event-kit/go/event_kit_api"
	"github.com/steadybit/extension-datadog/config"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"net/http"
	"time"
)

func RegisterEventListenerHandlers() {
	exthttp.RegisterHttpHandler("/events/experiment-started", onExperimentStarted)
	exthttp.RegisterHttpHandler("/events/experiment-completed", onExperimentCompleted)
}

type SendEventApi interface {
	SendEvent(ctx context.Context, eventBody StackStateIntakeRequest) (StackStateIntakeResponse, *http.Response, error)
}

func onExperimentStarted(w http.ResponseWriter, r *http.Request, body []byte) {
	event, err := parseBodyToEventRequestBody(body)
	if err != nil {
		exthttp.WriteError(w, extension_kit.ToError("Failed to decode event request body", err))
		return
	}


	request := StackStateIntakeRequest{
		Title: fmt.Sprintf("Experiment '%s' (execution ID %.0f) started", event.ExperimentExecution.Name, event.ExperimentExecution.ExecutionId),
		Text: fmt.Sprintf("%%%%%% \nThe chaos engineering experiment `%s` (execution %.0f) started.\n\nThe experiment is executed through [Steadybit](https://steadybit.com/?utm_campaign=extension-datadog&utm_source=extension-datadog-event).\n %%%%%%",
			event.ExperimentExecution.Name,
			event.ExperimentExecution.ExecutionId),
		Tags:           tags,
		SourceTypeName: extutil.Ptr("Steadybit"),
	}

	SendEvent(r.Context(), &config.Config, request)

	exthttp.WriteBody(w, "{}")
}

func onExperimentCompleted(w http.ResponseWriter, r *http.Request, body []byte) {
	event, err := parseBodyToEventRequestBody(body)
	if err != nil {
		exthttp.WriteError(w, extension_kit.ToError("Failed to decode event request body", err))
		return
	}
	tags := convertSteadybitEventToStackStateEventTags(event)
	if tags == nil {
		return
	}

	duration := event.ExperimentExecution.EndedTime.Sub(event.ExperimentExecution.PreparedTime)
	datadogEventBody := datadogV1.EventCreateRequest{
		Title: fmt.Sprintf("Experiment '%s' (execution ID %.0f) ended", event.ExperimentExecution.Name, event.ExperimentExecution.ExecutionId),
		Text: fmt.Sprintf("%%%%%% \nThe chaos engineering experiment `%s` (execution %.0f) ended with state `%s` after %.2f seconds.\n\nThe experiment was executed through [Steadybit](https://steadybit.com/?utm_campaign=extension-datadog&utm_source=extension-datadog-event).\n %%%%%%",
			event.ExperimentExecution.Name,
			event.ExperimentExecution.ExecutionId,
			event.ExperimentExecution.State,
			duration.Seconds()),
		Tags:           tags,
		SourceTypeName: extutil.Ptr("Steadybit"),
	}

	SendEvent(r.Context(), &config.Config, datadogEventBody)

	exthttp.WriteBody(w, "{}")
}


func parseBodyToEventRequestBody(body []byte) (event_kit_api.EventRequestBody, error) {
	var event event_kit_api.EventRequestBody
	err := json.Unmarshal(body, &event)
	return event, err
}

func SendEvent(ctx context.Context, api SendEventApi, datadogEventBody datadogV1.EventCreateRequest) {
	_, r, err := api.SendEvent(ctx, datadogEventBody)

	if err != nil {
		log.Err(err).Msgf("Failed to send Datadog event. Full response %v",
			r)
	} else if r.StatusCode != 202 && r.StatusCode != 200 {
		log.Error().Msgf("Datadog API responded with unexpected status code %d while sending Event. Full response: %v",
			r.StatusCode,
			r)
	}
}
