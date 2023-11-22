// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extservice

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_commons"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-stackstate/config"
	"strconv"
	"time"
)

type serviceDiscovery struct {
}

var (
	_ discovery_kit_sdk.TargetDescriber    = (*serviceDiscovery)(nil)
	_ discovery_kit_sdk.AttributeDescriber = (*serviceDiscovery)(nil)
)

func NewServiceDiscovery() discovery_kit_sdk.TargetDiscovery {
	discovery := &serviceDiscovery{}
	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsInterval(context.Background(), 1*time.Minute),
	)
}

func (d *serviceDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id:         serviceTargetType,
		RestrictTo: extutil.Ptr(discovery_kit_api.LEADER),
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("1m"),
		},
	}
}

func (d *serviceDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       serviceTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "StackState service", Other: "StackState services"},
		Category: extutil.Ptr("monitoring"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr(serviceIcon),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "k8s.service.name"},
				{Attribute: "k8s.namespace"},
				{Attribute: "k8s.cluster-name"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: "k8s.service.name",
					Direction: "ASC",
				},
			},
		},
	}
}

func (d *serviceDiscovery) DescribeAttributes() []discovery_kit_api.AttributeDescription {
	return []discovery_kit_api.AttributeDescription{
		{
			Attribute: "k8s.service.name",
			Label: discovery_kit_api.PluralLabel{
				One:   "Service",
				Other: "Services",
			},
		}, {
			Attribute: "k8s.namespace",
			Label: discovery_kit_api.PluralLabel{
				One:   "Namespace name",
				Other: "Namespace names",
			},
		}, {
			Attribute: "k8s.cluster-name",
			Label: discovery_kit_api.PluralLabel{
				One:   "Cluster name",
				Other: "Cluster names",
			},
		},
	}
}

func (d *serviceDiscovery) DiscoverTargets(ctx context.Context) ([]discovery_kit_api.Target, error) {
	return getAllServices(ctx, RestyClient), nil
}

func getAllServices(ctx context.Context, client *resty.Client) []discovery_kit_api.Target {
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
	return discovery_kit_commons.ApplyAttributeExcludes(result, config.Config.DiscoveryAttributesExcludesService)
}

func toService(service Component) discovery_kit_api.Target {
	clusterName := service.Properties.ClusterNameIdentifier[len("urn:cluster:/kubernetes:"):]
	namespace := service.Properties.NamespaceIdentifier[len(fmt.Sprintf("urn:kubernetes:/%v:namespace/", clusterName)):]
	attributes := make(map[string][]string)
	attributes["k8s.service.name"] = []string{service.Name}
	attributes["stackstate.service.id"] = []string{strconv.Itoa(service.Id)}
	attributes["k8s.namespace"] = []string{namespace}
	attributes["k8s.cluster-name"] = []string{clusterName}

	return discovery_kit_api.Target{
		Id:         strconv.Itoa(service.Id),
		Label:      service.Name,
		TargetType: serviceTargetType,
		Attributes: attributes,
	}
}
