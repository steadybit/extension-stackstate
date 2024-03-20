package extservice

import (
	"context"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPrepareExtractsState(t *testing.T) {
	// Given
	request := extutil.JsonMangle(action_kit_api.PrepareActionRequestBody{
		Config: map[string]interface{}{
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
		ExecutionContext: extutil.Ptr(action_kit_api.ExecutionContext{
			ExperimentUri: extutil.Ptr("<uri-to-experiment>"),
			ExecutionUri:  extutil.Ptr("<uri-to-execution>"),
		}),
	})
	action := ServiceStatusCheckAction{}
	state := action.NewEmptyState()

	// When
	result, err := action.Prepare(context.TODO(), &state, request)

	// Then
	require.Nil(t, result)
	require.Nil(t, err)
	require.Equal(t, state.ServiceId, "123")
	require.Equal(t, state.ServiceName, "test")
	require.Equal(t, state.ClusterName, "test-cluster")
	require.Equal(t, state.ExpectedStatus, "200")
	
}
