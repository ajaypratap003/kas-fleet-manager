/*
 * Connector Service Fleet Manager
 *
 * Connector Service Fleet Manager is a Rest API to manage connectors.
 *
 * API version: 0.1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

import (
	"time"
)

// ConnectorInstance struct for ConnectorInstance
type ConnectorInstance struct {
	Id                 string                           `json:"id,omitempty"`
	Kind               string                           `json:"kind,omitempty"`
	Href               string                           `json:"href,omitempty"`
	Owner              string                           `json:"owner,omitempty"`
	CreatedAt          time.Time                        `json:"created_at,omitempty"`
	ModifiedAt         time.Time                        `json:"modified_at,omitempty"`
	Name               string                           `json:"name"`
	ConnectorTypeId    string                           `json:"connector_type_id"`
	Channel            Channel                          `json:"channel,omitempty"`
	DeploymentLocation DeploymentLocation               `json:"deployment_location"`
	DesiredState       ConnectorDesiredState            `json:"desired_state"`
	ResourceVersion    int64                            `json:"resource_version,omitempty"`
	Kafka              KafkaConnectionSettings          `json:"kafka"`
	ServiceAccount     ServiceAccount                   `json:"service_account"`
	SchemaRegistry     SchemaRegistryConnectionSettings `json:"schema_registry,omitempty"`
	Connector          map[string]interface{}           `json:"connector"`
	Status             ConnectorInstanceStatusStatus    `json:"status,omitempty"`
}