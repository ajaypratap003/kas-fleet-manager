package config

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

type KafkaQuotaConfig struct {
	Type                         string
	AllowDeveloperInstance       bool
	MaxAllowedDeveloperInstances int
}

func NewKafkaQuotaConfig() *KafkaQuotaConfig {
	return &KafkaQuotaConfig{
		Type:                         api.QuotaManagementListQuotaType.String(),
		AllowDeveloperInstance:       true,
		MaxAllowedDeveloperInstances: 1,
	}
}
