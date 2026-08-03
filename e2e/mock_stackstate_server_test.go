/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

package e2e

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-stackstate/extservice"
)

type mockServer struct {
	http    *httptest.Server
	stateMu sync.RWMutex
	state   string
}

func (m *mockServer) setState(state string) {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	m.state = state
}

func (m *mockServer) getState() string {
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	return m.state
}

func createMockStackstateServer() *mockServer {
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		panic(fmt.Sprintf("httptest: failed to listen: %v", err))
	}
	mux := http.NewServeMux()

	server := httptest.Server{Listener: listener, Config: &http.Server{Handler: mux}}
	server.Start()
	log.Info().Str("url", server.URL).Msg("Started Mock-Server")

	mock := &mockServer{http: &server, state: "CLEAR"}
	mux.Handle("POST /snapshot", handler(mock.viewSnapshot))
	return mock
}

func handler[T any](getter func() T) http.Handler {
	return exthttp.PanicRecovery(exthttp.LogRequestWithDefaultLogLevel(exthttp.GetterAsHandler(getter), zerolog.DebugLevel))
}

func (m *mockServer) viewSnapshot() extservice.ViewSnapshotResponseWrapper {
	state := m.getState()
	if state == "STATUS-500" {
		panic("status 500")
	}
	return extservice.ViewSnapshotResponseWrapper{
		ViewSnapshotResponse: extservice.ViewSnapshotResponse{
			Components: []extservice.Component{
				{
					Id:          111,
					Name:        "eins-oelf",
					Identifiers: []string{"namespace-eins-elf/111"},
					Properties: extservice.Properties{
						ClusterNameIdentifier: "urn:cluster:/kubernetes:cluster-eins-elf",
						NamespaceIdentifier:   "urn:kubernetes:/cluster-eins-elf:namespace/namespace-eins-elf",
					},
					State: extservice.State{
						HealthState: state,
					},
				},
				{
					Id:          1310,
					Name:        "droelf-zehn",
					Identifiers: []string{"namespace-droelf-zehn/1310"},
					Properties: extservice.Properties{
						ClusterNameIdentifier: "urn:cluster:/kubernetes:cluster-droelf-zehn",
						NamespaceIdentifier:   "urn:kubernetes:/cluster-droelf-zehn:namespace/namespace-droelf-zehn",
					},
					State: extservice.State{
						HealthState: state,
					},
				},
			},
		},
	}
}
