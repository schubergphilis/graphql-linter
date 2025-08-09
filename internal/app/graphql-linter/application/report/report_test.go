package report

import (
	"testing"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	"github.com/stretchr/testify/assert"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/operationreport"
)

func TestStore_Summary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		schemaFiles []string
		totalErrors int
		passedFiles int
		allErrors   []models.DescriptionError
		want        Summary
	}{
		{
			name:        "all files pass",
			schemaFiles: []string{"a.graphql", "b.graphql"},
			totalErrors: 0,
			passedFiles: 2,
			allErrors:   nil,
			want: Summary{
				TotalFiles:                2,
				PassedFiles:               2,
				TotalErrors:               0,
				PercentPassed:             100.0,
				PercentageFilesWithErrors: 0.0,
				FilesWithAtLeastOneError:  0,
				AllErrors:                 nil,
			},
		},
		{
			name:        "some files fail",
			schemaFiles: []string{"a.graphql", "b.graphql"},
			totalErrors: 1,
			passedFiles: 1,
			allErrors: []models.DescriptionError{
				{FilePath: "a.graphql", LineNum: 1, Message: "error", LineContent: "foo"},
			},
			want: Summary{
				TotalFiles:                2,
				PassedFiles:               1,
				TotalErrors:               1,
				PercentPassed:             50.0,
				PercentageFilesWithErrors: 50.0,
				FilesWithAtLeastOneError:  1,
				AllErrors: []models.DescriptionError{
					{FilePath: "a.graphql", LineNum: 1, Message: "error", LineContent: "foo"},
				},
			},
		},
		{
			name:        "no files",
			schemaFiles: []string{},
			totalErrors: 0,
			passedFiles: 0,
			allErrors:   nil,
			want: Summary{
				TotalFiles:                0,
				PassedFiles:               0,
				TotalErrors:               0,
				PercentPassed:             0.0,
				PercentageFilesWithErrors: 0.0,
				FilesWithAtLeastOneError:  0,
				AllErrors:                 nil,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			summary := NewSummary(
				test.schemaFiles,
				test.totalErrors,
				test.passedFiles,
				test.allErrors,
			)

			if summary.TotalFiles != test.want.TotalFiles {
				t.Errorf("TotalFiles: got %d, want %d", summary.TotalFiles, test.want.TotalFiles)
			}

			if summary.PassedFiles != test.want.PassedFiles {
				t.Errorf("PassedFiles: got %d, want %d", summary.PassedFiles, test.want.PassedFiles)
			}

			if summary.TotalErrors != test.want.TotalErrors {
				t.Errorf("TotalErrors: got %d, want %d", summary.TotalErrors, test.want.TotalErrors)
			}

			if summary.PercentPassed != test.want.PercentPassed {
				t.Errorf(
					"PercentPassed: got %f, want %f",
					summary.PercentPassed,
					test.want.PercentPassed,
				)
			}

			if summary.PercentageFilesWithErrors != test.want.PercentageFilesWithErrors {
				t.Errorf(
					"PercentageFilesWithErrors: got %f, want %f",
					summary.PercentageFilesWithErrors,
					test.want.PercentageFilesWithErrors,
				)
			}

			if summary.FilesWithAtLeastOneError != test.want.FilesWithAtLeastOneError {
				t.Errorf(
					"FilesWithAtLeastOneError: got %d, want %d",
					summary.FilesWithAtLeastOneError,
					test.want.FilesWithAtLeastOneError,
				)
			}

			if !assert.Equal(t, test.want.AllErrors, summary.AllErrors) {
				t.Errorf("AllErrors: got %+v, want %+v", summary.AllErrors, test.want.AllErrors)
			}
		})
	}
}

func TestReportInternalErrors_Empty(t *testing.T) {
	t.Parallel()

	report := &operationreport.Report{}
	InternalErrors(report)
}

func TestReportExternalErrors_Empty(t *testing.T) {
	t.Parallel()

	report := &operationreport.Report{}
	ExternalErrors("foo", report, 1, 1)
}

func TestReportExternalErrorLocations_Nil(t *testing.T) {
	t.Parallel()

	lines := []string{"foo"}
	externalErr := operationreport.ExternalError{}
	reportExternalErrorLocations(lines, externalErr, 1, 1)
}

func TestReportContextLines_OutOfBounds(t *testing.T) {
	t.Parallel()

	lines := []string{"foo"}
	reportContextLines(lines, 100, 1, 1)
}
