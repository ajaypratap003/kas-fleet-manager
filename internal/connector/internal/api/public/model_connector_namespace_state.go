/*
 * Connector Service Fleet Manager
 *
 * Connector Service Fleet Manager is a Rest API to manage connectors.
 *
 * API version: 0.1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

// ConnectorNamespaceState the model 'ConnectorNamespaceState'
type ConnectorNamespaceState string

// List of ConnectorNamespaceState
const (
	CONNECTORNAMESPACESTATE_DISCONNECTED ConnectorNamespaceState = "disconnected"
	CONNECTORNAMESPACESTATE_READY        ConnectorNamespaceState = "ready"
	CONNECTORNAMESPACESTATE_DELETING     ConnectorNamespaceState = "deleting"
	CONNECTORNAMESPACESTATE_DELETED      ConnectorNamespaceState = "deleted"
)
