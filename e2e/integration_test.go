/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
	"context"
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_test/e2e"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_test/validate"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestWithMinikube(t *testing.T) {
	server := createMockStackstateServer()
	defer server.http.Close()
	split := strings.SplitAfter(server.http.URL, ":")
	port := split[len(split)-1]

	extlogging.InitZeroLog()

	extFactory := e2e.HelmExtensionFactory{
		Name: "extension-stackstate",
		Port: 8083,
		ExtraArgs: func(m *e2e.Minikube) []string {
			return []string{
				"--set", fmt.Sprintf("stackstate.apiBaseUrl=http://host.minikube.internal:%s", port),
				"--set", "logging.level=trace",
			}
		},
	}

	e2e.WithDefaultMinikube(t, &extFactory, []e2e.WithMinikubeTestCase{
		{
			Name: "validate discovery",
			Test: validateDiscovery,
		},
		{
			Name: "test discovery",
			Test: testDiscovery,
		},
		{
			Name: "service check meets expectations",
			Test: testServiceCheck(server, "CLEAR", "CLEAR", ""),
		},
		{
			Name: "service check fails expectations",
			Test: testServiceCheck(server, "DEVIATING", "CLEAR", action_kit_api.Failed),
		},
		{
			Name: "service check errors",
			Test: testServiceCheck(server, "STATUS-500", "CLEAR", action_kit_api.Failed),
		},
		{
			Name: "service check fails early by default",
			Test: testServiceCheckFailsEarlyByDefault(server),
		},
		{
			Name: "service check with fail early disabled fails at the end despite recovery",
			Test: testServiceCheckFailEarlyDisabledFailsAtEnd(server),
		},
		{
			Name: "service check at least once is not affected by fail early",
			Test: testServiceCheckAtLeastOnceIgnoresFailEarly(server),
		},
	})
}

type serviceCheckConfig struct {
	Duration        int    `json:"duration"`
	ExpectedStatus  string `json:"expectedStatus"`
	StatusCheckMode string `json:"statusCheckMode,omitempty"`
	FailEarly       *bool  `json:"failEarly,omitempty"`
}

func serviceCheckTarget() *action_kit_api.Target {
	return &action_kit_api.Target{
		Name: "111",
		Attributes: map[string][]string{
			"stackstate.service.id": {"111"},
			"k8s.service.name":      {"eins-oelf"},
			"k8s.cluster-name":      {"cluster-eins-elf"},
			"k8s.namespace":         {"namespace-eins-elf"},
		},
	}
}

func runServiceCheck(t *testing.T, e *e2e.Extension, config serviceCheckConfig) (time.Duration, error) {
	t.Helper()
	start := time.Now()
	action, err := e.RunAction("com.steadybit.extension_stackstate.service.check", serviceCheckTarget(), config, &action_kit_api.ExecutionContext{})
	require.NoError(t, err)
	defer func() { _ = action.Cancel() }()
	err = action.Wait()
	return time.Since(start), err
}

func testServiceCheck(server *mockServer, status, expectedStatus string, wantedActionStatus action_kit_api.ActionKitErrorStatus) func(t *testing.T, minikube *e2e.Minikube, e *e2e.Extension) {
	return func(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
		server.setState(status)
		_, err := runServiceCheck(t, e, serviceCheckConfig{Duration: 5_000, ExpectedStatus: expectedStatus})
		if wantedActionStatus == "" {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, fmt.Sprintf("[%s]", wantedActionStatus))
		}
	}
}

func testServiceCheckFailsEarlyByDefault(server *mockServer) func(t *testing.T, minikube *e2e.Minikube, e *e2e.Extension) {
	return func(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
		server.setState("DEVIATING")

		elapsed, err := runServiceCheck(t, e, serviceCheckConfig{Duration: 10_000, ExpectedStatus: "CLEAR"})

		require.ErrorContains(t, err, "has status 'DEVIATING'")
		require.Less(t, elapsed, 8*time.Second, "check with failEarly unset must fail on the first deviating status, well before the step duration")
	}
}

func testServiceCheckFailEarlyDisabledFailsAtEnd(server *mockServer) func(t *testing.T, minikube *e2e.Minikube, e *e2e.Extension) {
	return func(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
		server.setState("DEVIATING")
		// The service recovers mid-run; the check must still report the past deviation at the end.
		timer := time.AfterFunc(2500*time.Millisecond, func() { server.setState("CLEAR") })
		defer timer.Stop()

		failEarly := false
		elapsed, err := runServiceCheck(t, e, serviceCheckConfig{Duration: 6_000, ExpectedStatus: "CLEAR", FailEarly: &failEarly})

		require.ErrorContains(t, err, "had status 'DEVIATING'")
		require.GreaterOrEqual(t, elapsed, 6*time.Second, "check with failEarly disabled must run for the whole step duration")
	}
}

func testServiceCheckAtLeastOnceIgnoresFailEarly(server *mockServer) func(t *testing.T, minikube *e2e.Minikube, e *e2e.Extension) {
	return func(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
		server.setState("DEVIATING")
		// The expected status only appears mid-run: failEarly must not make 'at least once' fail beforehand.
		timer := time.AfterFunc(2500*time.Millisecond, func() { server.setState("CLEAR") })
		defer timer.Stop()

		failEarly := true
		_, err := runServiceCheck(t, e, serviceCheckConfig{Duration: 6_000, ExpectedStatus: "CLEAR", StatusCheckMode: "atLeastOnce", FailEarly: &failEarly})

		require.NoError(t, err)
	}
}

func validateDiscovery(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
	assert.NoError(t, validate.ValidateEndpointReferences("/", e.Client))
}

func testDiscovery(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	target, err := e2e.PollForTarget(ctx, e, "com.steadybit.extension_stackstate.service", func(target discovery_kit_api.Target) bool {
		return e2e.HasAttribute(target, "stackstate.service.id", "111")
	})
	require.NoError(t, err)
	assert.Equal(t, target.TargetType, "com.steadybit.extension_stackstate.service")
	assert.Equal(t, target.Attributes["k8s.service.name"], []string{"eins-oelf"})
	assert.Equal(t, target.Attributes["k8s.cluster-name"], []string{"cluster-eins-elf"})
	assert.Equal(t, target.Attributes["k8s.namespace"], []string{"namespace-eins-elf"})
}
