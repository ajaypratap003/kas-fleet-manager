/*
 * Connector Service Fleet Manager
 *
 * Connector Service Fleet Manager is a Rest API to manage connectors.
 *
 * API version: 0.1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

// ConnectorInstanceStatusStatus struct for ConnectorInstanceStatusStatus
type ConnectorInstanceStatusStatus struct {
	State ConnectorState `json:"state,omitempty"`
	Error string         `json:"error,omitempty"`
}