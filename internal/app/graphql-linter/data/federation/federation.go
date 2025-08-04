package federation

import (
	"strings"

	log "github.com/sirupsen/logrus"
)

// ValidateFederationSchema checks if the schema string contains federation directives.
func ValidateFederationSchema(schema string) bool {
	// This is a placeholder implementation. Replace with actual federation validation logic as needed.
	if strings.Contains(schema, "@key") || strings.Contains(schema, "@external") {
		log.Debug("Federation directives detected in schema.")
		return true
	}
	log.Debug("No federation directives detected in schema.")
	return true // or false if you want to fail when federation directives are missing
}
