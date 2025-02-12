// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package extservice

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
)

const (
	serviceTargetType          = "com.steadybit.extension_stackstate.service"
	serviceIcon                = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA5Ny4yNyA5Ni42MSI+PGcgZmlsbD0iY3VycmVudENvbG9yIj48cGF0aCBkPSJNMTcuOTUgMjkuN2wzMC43OS0xNy43Mkw3OS41MyAyOS43IDUwLjY0IDQ2LjI5Yy0xLjE3LjY5LTIuNjUuNjktMy44MSAwTDE3Ljk1IDI5Ljd6Ii8+PHBhdGggZD0iTTQ2Ljg0IDUzLjY0TDI2LjcxIDQyLjA2bC04Ljc2IDUuMDYgMjguODggMTYuNTljMS4xNy42OSAyLjY1LjY5IDMuODEgMGwyOC44OC0xNi41OS04Ljc2LTUuMDYtMjAuMTMgMTEuNThjLTEuMTcuNjktMi42NS42OS0zLjgxIDB6Ii8+PHBhdGggZD0iTTQ2Ljg0IDcxLjQ1TDI2LjcxIDU5Ljg3bC04Ljc2IDUuMDYgMjguODggMTYuNTljMS4xNy42OSAyLjY1LjY5IDMuODEgMGwyOC44OC0xNi41OS04Ljc2LTUuMDYtMjAuMDggMTEuNThjLTEuMjEuNjktMi42OS42OS0zLjg1IDB6Ii8+PGc+PHBhdGggZD0iTTAgNDguMzJjMCA4LjU2IDIuMjUgMTYuNTkgNi4xNiAyMy41Nmw1LjQ2LTMuMTZ2LTQuOTdjMC0xLjM0Ljc0LTIuNiAxLjkxLTMuMjhsNi45LTMuOTgtNi42OC0zLjg1Yy0xLjE3LS42OS0xLjkxLTEuOTUtMS45MS0zLjI4VjQ1LjljMC0xLjM0LjctMi42NCAxLjkxLTMuMjhsNi42NC0zLjg5LTYuNTktMy44Yy0xLjE3LS42OS0xLjkxLTEuOTUtMS45MS0zLjI4di0zLjAyYzAtMS4zOS43NC0yLjY0IDEuOTEtMy4zM0w0NS4yMyA3LjI3Vi4xM0MxOS45OSAxLjgxIDAgMjIuNzggMCA0OC4zMnpNOTcuMjcgNDguMjRjMCA4LjU2LTIuMjUgMTYuNTktNi4xNiAyMy41NmwtNS40Ni0zLjE2di00Ljk3YzAtMS4zNC0uNzQtMi42LTEuOTEtMy4yOGwtNi45LTMuOTggNi42OC0zLjg1YzEuMTctLjY5IDEuOTEtMS45NSAxLjkxLTMuMjh2LTMuNDZjMC0xLjM0LS43LTIuNjQtMS45MS0zLjI4bC02LjY0LTMuODkgNi41OS0zLjhjMS4xNy0uNjkgMS45MS0xLjk1IDEuOTEtMy4yOHYtMy4wMmMwLTEuMzktLjc0LTIuNjQtMS45MS0zLjMzTDUyLjA0IDcuMTdWMGMyNS4yNCAxLjY5IDQ1LjIzIDIyLjY5IDQ1LjIzIDQ4LjI0ek00OC42MSA5Ni42MWMxNS45NiAwIDMwLjE0LTcuNjkgMzguOTktMTkuNTNsLTguMzMtNC43NUw1MC42OSA4OC43Yy0xLjE3LjY5LTIuNjUuNjktMy44MSAwTDE4IDcyLjE1bC01LjMgMy4wMi0uNi4zNC0yLjU2IDEuNDdjOC44OSAxMS44OSAyMy4wNyAxOS42MyAzOS4wNyAxOS42M3oiLz48L2c+PC9nPjwvc3ZnPg=="
	statusCheckModeAtLeastOnce = "atLeastOnce"
	statusCheckModeAllTheTime  = "allTheTime"

	attributeServiceId     = "stackstate.service.id"
	attributeK8ServiceName = "k8s.service.name"
	attributeK8ClusterName = "k8s.cluster-name"
	attributeK8Namespace   = "k8s.namespace"
	attributeState         = "state"
	attributeTooltip       = "tooltip"
	attributeUrl           = "url"
)

var Client *StackStateHttpClient

type StackStateHttpClient struct {
	Client *resty.Client
}

func (s *StackStateHttpClient) GetServiceSnapshot(ctx context.Context, serviceId string) (*resty.Response, ViewSnapshotResponseWrapper, error) {
	return s.executeSnapshotQuery(ctx, fmt.Sprintf("(id = \\\"%s\\\")", serviceId))
}

func (s *StackStateHttpClient) GetServiceSnapshots(ctx context.Context) (*resty.Response, ViewSnapshotResponseWrapper, error) {
	return s.executeSnapshotQuery(ctx, "(type = \\\"service\\\")")
}

func (s *StackStateHttpClient) executeSnapshotQuery(ctx context.Context, query string) (*resty.Response, ViewSnapshotResponseWrapper, error) {
	requestBody := fmt.Sprintf(`{
    "_type": "ViewSnapshotRequest",
    "query": "%v",
    "queryVersion": "0.0.1",
    "metadata": {
        "_type": "QueryMetadata",
        "groupingEnabled": false,
        "showIndirectRelations": false,
        "minGroupSize": 0,
        "groupedByLayer": false,
        "groupedByDomain": false,
        "groupedByRelation": false,
        "showCause": "NONE",
        "autoGrouping": false,
        "connectedComponents": false,
        "neighboringComponents": false,
        "showFullComponent": false
    }
  }`, query)
	var stackStateResponse ViewSnapshotResponseWrapper
	response, err := s.Client.R().
		SetContext(ctx).
		SetBody([]byte(requestBody)).
		SetResult(&stackStateResponse).
		Post("/snapshot")
	return response, stackStateResponse, err
}
