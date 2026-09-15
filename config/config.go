/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package config

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
	"strings"
)

// Specification is the configuration specification for the extension. Configuration values can be applied
// through environment variables. Learn more through the documentation of the envconfig package.
// https://github.com/kelseyhightower/envconfig
type Specification struct {
	ServiceToken                       string   `json:"serviceToken" split_words:"true" required:"true"`
	ApiBaseUrl                         string   `json:"apiBaseUrl" split_words:"true" required:"true"`
	DiscoveryAttributesExcludesService []string `json:"discoveryAttributesExcludesService" split_words:"true" required:"false"`
}

var (
	Config Specification
)

func ParseConfiguration() {
	err := envconfig.Process("steadybit_extension", &Config)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to parse configuration from environment.")
	}
}

func ValidateConfiguration() {
	// envconfig's `required:"true"` only checks that the variable is *set*: an empty
	// value satisfies it, so the extension would start with a blank configuration and
	// fail much later against the target system. Reject blank values here instead.
	if strings.TrimSpace(Config.ServiceToken) == "" {
		log.Fatal().Msg("STEADYBIT_EXTENSION_SERVICE_TOKEN must not be empty.")
	}
	if strings.TrimSpace(Config.ApiBaseUrl) == "" {
		log.Fatal().Msg("STEADYBIT_EXTENSION_API_BASE_URL must not be empty.")
	}
}
