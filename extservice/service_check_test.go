package extservice

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-stackstate/config"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

type getSnapshotApiMock struct {
	mock.Mock
}

func (m *getSnapshotApiMock) GetServiceSnapshot(ctx context.Context, serviceId string) (*resty.Response, ViewSnapshotResponseWrapper, error) {
	args := m.Called(ctx, serviceId)
	return args.Get(0).(*resty.Response), args.Get(1).(ViewSnapshotResponseWrapper), args.Error(2)
}

var action = NewServiceStatusCheckAction()

func TestServiceCheck(t *testing.T) {

	config.Config.ApiBaseUrl = "http://integration-test.invalid/api"

	t.Run("Prepare extracts state", func(t *testing.T) {
		// Given
		request := extutil.JsonMangle(action_kit_api.PrepareActionRequestBody{
			Config: map[string]any{
				"duration":        1000 * 60,
				"expectedStatus":  "200",
				"statusCheckMode": "atLeastOnce",
			},
			Target: &action_kit_api.Target{
				Attributes: map[string][]string{
					"stackstate.service.id": {"123"},
					"k8s.service.name":      {"test"},
					"k8s.cluster-name":      {"test-cluster"},
				},
			},
			ExecutionContext: new(action_kit_api.ExecutionContext{
				ExperimentUri: new("<uri-to-experiment>"),
				ExecutionUri:  new("<uri-to-execution>"),
			}),
		})
		state := serviceCheckState(statusCheckModeAtLeastOnce)

		// When
		result, err := action.Prepare(context.TODO(), &state, request)

		// Then
		require.Nil(t, result)
		require.NoError(t, err)
		require.Equal(t, state.ServiceId, "123")
		require.Equal(t, state.ServiceName, "test")
		require.Equal(t, state.ClusterName, "test-cluster")
		require.Equal(t, state.ExpectedStatus, "200")
		require.Equal(t, state.StatusCheckMode, statusCheckModeAtLeastOnce)
		require.Equal(t, state.StatusCheckSuccess, false)
	})

	t.Run("Prepare extracts state without statusCheckMode", func(t *testing.T) {
		// Given
		request := extutil.JsonMangle(action_kit_api.PrepareActionRequestBody{
			Config: map[string]any{
				"duration":       1000 * 60,
				"expectedStatus": "200",
			},
			Target: &action_kit_api.Target{
				Attributes: map[string][]string{
					"stackstate.service.id": {"123"},
					"k8s.service.name":      {"test"},
					"k8s.cluster-name":      {"test-cluster"},
				},
			},
			ExecutionContext: new(action_kit_api.ExecutionContext{
				ExperimentUri: new("<uri-to-experiment>"),
				ExecutionUri:  new("<uri-to-execution>"),
			}),
		})
		state := serviceCheckState(statusCheckModeAllTheTime)

		// When
		result, err := action.Prepare(context.TODO(), &state, request)

		// Then
		require.Nil(t, result)
		require.NoError(t, err)
		require.Equal(t, state.ServiceId, "123")
		require.Equal(t, state.ServiceName, "test")
		require.Equal(t, state.ClusterName, "test-cluster")
		require.Equal(t, state.ExpectedStatus, "200")
		require.Equal(t, state.StatusCheckMode, statusCheckModeAllTheTime)
		require.Equal(t, state.StatusCheckSuccess, true)
	})

	t.Run("status allTheTime success", func(t *testing.T) {
		state := serviceCheckState(statusCheckModeAllTheTime)
		response := apiResponseWithStatus(200)
		wrapper := serviceResponseWithState("CLEAR")
		mockedApi := new(getSnapshotApiMock)
		mockedApi.On("GetServiceSnapshot", mock.Anything, mock.Anything).Return(response, wrapper, nil)

		status, err := MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
		require.NoError(t, err)
		require.NotNil(t, status)
		require.False(t, status.Completed)
		require.Nil(t, status.Error)
		require.Equal(t, (*status.Metrics)[0].Metric["state"], "success")

		state.End = time.Now().Add(-1 * time.Hour)
		status, err = MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
		require.NoError(t, err)
		require.NotNil(t, status)
		require.True(t, status.Completed)
		require.Nil(t, status.Error)
	})

	t.Run("status allTheTime failure", func(t *testing.T) {
		state := serviceCheckState(statusCheckModeAllTheTime)
		response := apiResponseWithStatus(200)
		wrapper := serviceResponseWithState("CLEAR")
		mockedApi := new(getSnapshotApiMock)
		mockedApi.On("GetServiceSnapshot", mock.Anything, mock.Anything).Return(response, wrapper, nil)

		status, err := MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
		require.NoError(t, err)
		require.NotNil(t, status)
		require.False(t, status.Completed)
		require.Nil(t, status.Error)
		require.Equal(t, (*status.Metrics)[0].Metric["state"], "success")

		mockedApi.On("GetServiceSnapshot", mock.Anything, mock.Anything).Unset()
		mockedApi.On("GetServiceSnapshot", mock.Anything, mock.Anything).Return(response, serviceResponseWithState("DEVIATING"), nil)

		status, err = MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
		require.NoError(t, err)
		require.NotNil(t, status)
		require.False(t, status.Completed)
		require.NotNil(t, status.Error)
		require.Equal(t, (*status.Metrics)[0].Metric["state"], "warn")

		state.End = time.Now().Add(-1 * time.Hour)
		status, err = MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
		require.NoError(t, err)
		require.NotNil(t, status)
		require.True(t, status.Completed)
		require.NotNil(t, status.Error)
		require.Equal(t, (*status.Metrics)[0].Metric["state"], "warn")
	})

	t.Run("status allTheTime fail at end", func(t *testing.T) {
		state := serviceCheckState(statusCheckModeAllTheTime)
		state.FailEarly = false
		mockedApi := new(getSnapshotApiMock)
		mockedApi.On("GetServiceSnapshot", mock.Anything, mock.Anything).Return(apiResponseWithStatus(200), serviceResponseWithState("DEVIATING"), nil)

		// Deviation observed but time not up: must not fail early, deviation is remembered.
		status, err := MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
		require.NoError(t, err)
		require.False(t, status.Completed)
		require.Nil(t, status.Error)
		require.NotEmpty(t, state.DeviationTitle)

		// Time is up: the remembered deviation is reported with the past-tense message.
		state.End = time.Now().Add(-1 * time.Hour)
		status, err = MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
		require.NoError(t, err)
		require.True(t, status.Completed)
		require.NotNil(t, status.Error)
		require.Contains(t, status.Error.Title, "had status")
	})

	t.Run("status atLeastOnce success", func(t *testing.T) {
		state := serviceCheckState(statusCheckModeAtLeastOnce)
		response := apiResponseWithStatus(200)
		wrapper := serviceResponseWithState("UNKNOWN")
		mockedApi := new(getSnapshotApiMock)
		mockedApi.On("GetServiceSnapshot", mock.Anything, mock.Anything).Return(response, wrapper, nil)

		status, err := MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
		require.NoError(t, err)
		require.NotNil(t, status)
		require.False(t, status.Completed)
		require.Nil(t, status.Error)
		require.Equal(t, (*status.Metrics)[0].Metric["state"], "warn")

		mockedApi.On("GetServiceSnapshot", mock.Anything, mock.Anything).Unset()
		mockedApi.On("GetServiceSnapshot", mock.Anything, mock.Anything).Return(response, serviceResponseWithState("CLEAR"), nil)

		status, err = MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
		require.NoError(t, err)
		require.NotNil(t, status)
		require.False(t, status.Completed)
		require.Nil(t, status.Error)
		require.Equal(t, (*status.Metrics)[0].Metric["state"], "success")

		state.End = time.Now().Add(-1 * time.Hour)
		status, err = MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
		require.NoError(t, err)
		require.NotNil(t, status)
		require.True(t, status.Completed)
		require.Nil(t, status.Error)
		require.Equal(t, (*status.Metrics)[0].Metric["state"], "success")
	})

	t.Run("status atLeastOnce failure", func(t *testing.T) {
		state := serviceCheckState(statusCheckModeAtLeastOnce)
		response := apiResponseWithStatus(200)
		wrapper := serviceResponseWithState("CRITICAL")
		mockedApi := new(getSnapshotApiMock)
		mockedApi.On("GetServiceSnapshot", mock.Anything, mock.Anything).Return(response, wrapper, nil)

		status, err := MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
		require.NoError(t, err)
		require.NotNil(t, status)
		require.False(t, status.Completed)
		require.Nil(t, status.Error)
		require.Equal(t, (*status.Metrics)[0].Metric["state"], "danger")

		state.End = time.Now().Add(-1 * time.Hour)
		status, err = MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
		require.NoError(t, err)
		require.NotNil(t, status)
		require.True(t, status.Completed)
		require.NotNil(t, status.Error)
		require.Equal(t, (*status.Metrics)[0].Metric["state"], "danger")
	})

	t.Run("status error on api maps to UNKNOWN", func(t *testing.T) {
		state := serviceCheckState(statusCheckModeAtLeastOnce)
		response := apiResponseWithStatus(500)
		wrapper := serviceResponseWithState("DEVIATING")
		mockedApi := new(getSnapshotApiMock)
		mockedApi.On("GetServiceSnapshot", mock.Anything, mock.Anything).Return(response, wrapper, nil)

		status, err := MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
		require.NoError(t, err)
		require.NotNil(t, status)
		require.False(t, status.Completed)
		require.Equal(t, (*status.Metrics)[0].Metric["state"], "warn")
	})

	t.Run("connection issue results in error", func(t *testing.T) {
		state := serviceCheckState(statusCheckModeAtLeastOnce)
		response := apiResponseWithStatus(500)
		wrapper := serviceResponseWithState("DEVIATING")
		mockedApi := new(getSnapshotApiMock)
		err := errors.New("Test error")
		mockedApi.On("GetServiceSnapshot", mock.Anything, mock.Anything).Return(response, wrapper, err)

		status, err := MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
		require.Error(t, err)
		require.Nil(t, status)
	})
}

func TestServiceCheckFailEarly(t *testing.T) {

	config.Config.ApiBaseUrl = "http://integration-test.invalid/api"

	prepareRequest := func(cfg map[string]any) action_kit_api.PrepareActionRequestBody {
		return extutil.JsonMangle(action_kit_api.PrepareActionRequestBody{
			Config: cfg,
			Target: &action_kit_api.Target{
				Attributes: map[string][]string{
					"stackstate.service.id": {"123"},
					"k8s.service.name":      {"test"},
					"k8s.cluster-name":      {"test-cluster"},
				},
			},
			ExecutionContext: extutil.Ptr(action_kit_api.ExecutionContext{
				ExperimentUri: extutil.Ptr("<uri-to-experiment>"),
				ExecutionUri:  extutil.Ptr("<uri-to-execution>"),
			}),
		})
	}

	t.Run("Describe declares failEarly as advanced with default true", func(t *testing.T) {
		description := action.Describe()
		var failEarly *action_kit_api.ActionParameter
		for _, p := range description.Parameters {
			if p.Name == "failEarly" {
				failEarly = &p
				break
			}
		}
		require.NotNil(t, failEarly)
		require.Equal(t, action_kit_api.ActionParameterTypeBoolean, failEarly.Type)
		require.Equal(t, "true", *failEarly.DefaultValue)
		require.True(t, *failEarly.Advanced)
	})

	t.Run("Prepare defaults failEarly to true when parameter is absent (non-breaking)", func(t *testing.T) {
		state := action.NewEmptyState()
		_, err := action.Prepare(context.TODO(), &state, prepareRequest(map[string]any{
			"duration":        1000 * 60,
			"expectedStatus":  "CLEAR",
			"statusCheckMode": statusCheckModeAllTheTime,
		}))
		require.NoError(t, err)
		require.True(t, state.FailEarly)
	})

	t.Run("Prepare extracts failEarly false", func(t *testing.T) {
		state := action.NewEmptyState()
		_, err := action.Prepare(context.TODO(), &state, prepareRequest(map[string]any{
			"duration":        1000 * 60,
			"expectedStatus":  "CLEAR",
			"statusCheckMode": statusCheckModeAllTheTime,
			"failEarly":       false,
		}))
		require.NoError(t, err)
		require.False(t, state.FailEarly)
	})

	t.Run("allTheTime fail at end reports deviation even after recovery", func(t *testing.T) {
		state := serviceCheckState(statusCheckModeAllTheTime)
		state.FailEarly = false
		mockedApi := new(getSnapshotApiMock)
		mockedApi.On("GetServiceSnapshot", mock.Anything, mock.Anything).Return(apiResponseWithStatus(200), serviceResponseWithState("DEVIATING"), nil)

		// Deviation observed mid-run: no early failure, metric still emitted.
		status, err := MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
		require.NoError(t, err)
		require.False(t, status.Completed)
		require.Nil(t, status.Error)
		require.Equal(t, "warn", (*status.Metrics)[0].Metric["state"])

		// Service recovers before the end of the step.
		mockedApi.On("GetServiceSnapshot", mock.Anything, mock.Anything).Unset()
		mockedApi.On("GetServiceSnapshot", mock.Anything, mock.Anything).Return(apiResponseWithStatus(200), serviceResponseWithState("CLEAR"), nil)

		status, err = MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
		require.NoError(t, err)
		require.False(t, status.Completed)
		require.Nil(t, status.Error)
		require.Equal(t, "success", (*status.Metrics)[0].Metric["state"])

		// Time is up: the past deviation is still reported.
		state.End = time.Now().Add(-1 * time.Hour)
		status, err = MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
		require.NoError(t, err)
		require.True(t, status.Completed)
		require.NotNil(t, status.Error)
		require.Contains(t, status.Error.Title, "had status 'DEVIATING'")
	})

	t.Run("allTheTime fail at end passes without deviation", func(t *testing.T) {
		state := serviceCheckState(statusCheckModeAllTheTime)
		state.FailEarly = false
		mockedApi := new(getSnapshotApiMock)
		mockedApi.On("GetServiceSnapshot", mock.Anything, mock.Anything).Return(apiResponseWithStatus(200), serviceResponseWithState("CLEAR"), nil)

		status, err := MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
		require.NoError(t, err)
		require.False(t, status.Completed)
		require.Nil(t, status.Error)

		state.End = time.Now().Add(-1 * time.Hour)
		status, err = MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
		require.NoError(t, err)
		require.True(t, status.Completed)
		require.Nil(t, status.Error)
	})

	for _, failEarly := range []bool{true, false} {
		t.Run(fmt.Sprintf("atLeastOnce never fails early regardless of failEarly=%v", failEarly), func(t *testing.T) {
			state := serviceCheckState(statusCheckModeAtLeastOnce)
			state.FailEarly = failEarly
			mockedApi := new(getSnapshotApiMock)
			mockedApi.On("GetServiceSnapshot", mock.Anything, mock.Anything).Return(apiResponseWithStatus(200), serviceResponseWithState("DEVIATING"), nil)

			// Deviating mid-run: must never fail early - the expected status may still appear later.
			status, err := MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
			require.NoError(t, err)
			require.False(t, status.Completed)
			require.Nil(t, status.Error)

			// Expected status appears later: the check succeeds at the end.
			mockedApi.On("GetServiceSnapshot", mock.Anything, mock.Anything).Unset()
			mockedApi.On("GetServiceSnapshot", mock.Anything, mock.Anything).Return(apiResponseWithStatus(200), serviceResponseWithState("CLEAR"), nil)

			status, err = MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
			require.NoError(t, err)
			require.Nil(t, status.Error)

			state.End = time.Now().Add(-1 * time.Hour)
			status, err = MonitorStatusCheckStatus(context.TODO(), &state, mockedApi)
			require.NoError(t, err)
			require.True(t, status.Completed)
			require.Nil(t, status.Error)
		})
	}
}

func serviceCheckState(mode string) ServiceStatusCheckState {
	state := action.NewEmptyState()
	state.ServiceId = "123"
	state.ServiceName = "test"
	state.ClusterName = "test-cluster"
	state.ExpectedStatus = "CLEAR"
	state.StatusCheckMode = mode
	state.StatusCheckSuccess = mode == statusCheckModeAllTheTime
	state.FailEarly = true // matches the production default; 'All the time' fails fast
	state.End = time.Now().Add(1 * time.Hour)
	return state
}

func apiResponseWithStatus(status int) *resty.Response {
	return &resty.Response{
		Request: nil,
		RawResponse: &http.Response{
			StatusCode: status,
		},
	}
}

func serviceResponseWithState(state string) ViewSnapshotResponseWrapper {
	return ViewSnapshotResponseWrapper{
		ViewSnapshotResponse: ViewSnapshotResponse{
			Components: []Component{
				{
					Id:   1,
					Name: "service1",
					State: State{
						HealthState: state,
					},
					Identifiers: []string{
						"identifier1", "identifier2",
					},
				},
			},
		},
	}
}
