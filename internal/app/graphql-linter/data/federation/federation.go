package federation

import (
	"fmt"
	"log/slog"

	"github.com/wundergraph/graphql-go-tools/v2/pkg/federation"
)

func ValidateFederationSchema(filteredSchema string) bool {
	_, err := federation.BuildFederationSchema(filteredSchema, filteredSchema)
	if err != nil {
		slog.Info(fmt.Sprintf("Federation schema build failed: %v", err))

		return false
	}

	slog.Debug("Federation schema validation passed")

	return true
}
