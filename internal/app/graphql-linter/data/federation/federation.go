package federation

import (
	"fmt"
	"log/slog"

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
		slog.Info(fmt.Sprintf("Federation schema build failed: %v", federationErr))

		return false
	}

	_ = federationSchema

	if report.HasErrors() {
		slog.Error("Federation validation errors:")

		for _, internalErr := range report.InternalErrors {
			slog.Error(fmt.Sprintf("  - %v", internalErr))
		}

		for _, externalErr := range report.ExternalErrors {
			slog.Error("  - " + externalErr.Message)
		}

		return false
	}

	slog.Debug("Federation schema validation passed")

	return true
}
