package federation

import (
	log "github.com/sirupsen/logrus"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/federation"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/operationreport"
)

func ValidateFederationSchema(filteredSchema string) bool {
	var report operationreport.Report

	federationSchema, federationErr := federation.BuildFederationSchema(
		filteredSchema,
		filteredSchema,
	)
	if federationErr != nil {
		log.Infof("Federation schema build failed: %v\n", federationErr)

		return false
	}

	_ = federationSchema

	if report.HasErrors() {
		log.Error("Federation validation errors:")

		for _, internalErr := range report.InternalErrors {
			log.Errorf("  - %v\n", internalErr)
		}

		for _, externalErr := range report.ExternalErrors {
			log.Errorf("  - %s\n", externalErr.Message)
		}

		return false
	}

	log.Debug("Federation schema validation passed")

	return true
}
