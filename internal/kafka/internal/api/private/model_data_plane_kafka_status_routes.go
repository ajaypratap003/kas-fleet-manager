/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager APIs that are used by internal services e.g kas-fleetshard operators.
 *
 * API version: 1.2.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// DataPlaneKafkaStatusRoutes struct for DataPlaneKafkaStatusRoutes
type DataPlaneKafkaStatusRoutes struct {
	Name   string `json:"name,omitempty"`
	Prefix string `json:"prefix,omitempty"`
	Router string `json:"router,omitempty"`
}
