// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extservice

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"net/http"
	"strconv"
)

func RegisterServiceDiscoveryHandlers() {
	exthttp.RegisterHttpHandler("/service/discovery", exthttp.GetterAsHandler(getServiceDiscoveryDescription))
	exthttp.RegisterHttpHandler("/service/discovery/target-description", exthttp.GetterAsHandler(getServiceTargetDescription))
	exthttp.RegisterHttpHandler("/service/discovery/attribute-descriptions", exthttp.GetterAsHandler(getServiceAttributeDescriptions))
	exthttp.RegisterHttpHandler("/service/discovery/discovered-targets", getServiceDiscoveryResults)
}

func GetDiscoveryList() discovery_kit_api.DiscoveryList {
	return discovery_kit_api.DiscoveryList{
		Discoveries: []discovery_kit_api.DescribingEndpointReference{
			{
				Method: "GET",
				Path:   "/service/discovery",
			},
		},
		TargetTypes: []discovery_kit_api.DescribingEndpointReference{
			{
				Method: "GET",
				Path:   "/service/discovery/target-description",
			},
		},
		TargetAttributes: []discovery_kit_api.DescribingEndpointReference{
			{
				Method: "GET",
				Path:   "/service/discovery/attribute-descriptions",
			},
		},
	}
}

func getServiceDiscoveryDescription() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id:         serviceTargetType,
		RestrictTo: extutil.Ptr(discovery_kit_api.LEADER),
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			Method:       "GET",
			Path:         "/service/discovery/discovered-targets",
			CallInterval: extutil.Ptr("5m"),
		},
	}
}

func getServiceTargetDescription() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       serviceTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "StackState service", Other: "StackState service"},
		Category: extutil.Ptr("monitoring"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr(serviceIcon),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "stackstate.service"},
				{Attribute: "stackstate.namespace"},
				{Attribute: "stackstate.cluster-name"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: "stackstate.service",
					Direction: "ASC",
				},
			},
		},
	}
}

func getServiceAttributeDescriptions() discovery_kit_api.AttributeDescriptions {
	return discovery_kit_api.AttributeDescriptions{
		Attributes: []discovery_kit_api.AttributeDescription{
			{
				Attribute: "stackstate.service",
				Label: discovery_kit_api.PluralLabel{
					One:   "Service",
					Other: "Services",
				},
			}, {
				Attribute: "stackstate.namespace",
				Label: discovery_kit_api.PluralLabel{
					One:   "Namespace name",
					Other: "Namespace names",
				},
			}, {
				Attribute: "stackstate.cluster-name",
				Label: discovery_kit_api.PluralLabel{
					One:   "Cluster name",
					Other: "Cluster names",
				},
			},
		},
	}
}

func getServiceDiscoveryResults(w http.ResponseWriter, r *http.Request, _ []byte) {
	exthttp.WriteBody(w, discovery_kit_api.DiscoveredTargets{Targets: GetAllServices(r.Context(), RestyClient)})
}

func GetAllServices(ctx context.Context, client *resty.Client) []discovery_kit_api.Target {
	result := make([]discovery_kit_api.Target, 0, 500)

	var stackStateResponse ViewSnapshotResponseWrapper
	res, err := client.R().
		SetContext(ctx).
		SetBody([]byte(`{
    "_type": "ViewSnapshotRequest",
    "query": "(type = \"service\")",
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
        "autoGrouping": false
    }
  }`)).
		SetResult(&stackStateResponse).
		Post("/snapshot")

	if err != nil {
		log.Err(err).Msgf("Failed to retrieve service states from Stack State. Full response: %v", res.String())
		return result
	}

	if res.StatusCode() != 200 {
		log.Error().Msgf("StackState API responded with unexpected status code %d while retrieving service states. Full response: %v",
			res.StatusCode(),
			res.String())
		return result
	}

	if len(stackStateResponse.ViewSnapshotResponse.Components) > 0 {
		for _, component := range stackStateResponse.ViewSnapshotResponse.Components {
			result = append(result, toService(component))
		}
	}
	return result
}

func toService(service Component) discovery_kit_api.Target {
	clusterName := service.Properties.ClusterNameIdentifier[len("urn:cluster:/kubernetes:"):]
	namespace := service.Properties.NamespaceIdentifier[len(fmt.Sprintf("urn:kubernetes:/%v:namespace/", clusterName)):]
	attributes := make(map[string][]string)
	attributes["stackstate.service"] = []string{service.Name}
	attributes["stackstate.service.id"] = []string{strconv.Itoa(service.Id)}
	attributes["stackstate.namespace"] = []string{namespace}
	attributes["stackstate.cluster-name"] = []string{clusterName}

	return discovery_kit_api.Target{
		Id:         strconv.Itoa(service.Id),
		Label:      service.Name,
		TargetType: serviceTargetType,
		Attributes: attributes,
	}
}
