/*
 * Connector Service Fleet Manager
 *
 * Connector Service Fleet Manager is a Rest API to manage connectors.
 *
 * API version: 0.1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

// ConnectorList struct for ConnectorList
type ConnectorList struct {
	Kind  string      `json:"kind"`
	Page  int32       `json:"page"`
	Size  int32       `json:"size"`
	Total int32       `json:"total"`
	Items []Connector `json:"items"`
}