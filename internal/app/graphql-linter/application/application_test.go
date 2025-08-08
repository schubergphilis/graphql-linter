package application

import (
	"reflect"
	"runtime/debug"
	"testing"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/application/mocks"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data"
	"github.com/stretchr/testify/assert"
)

func TestExecute_Version(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		versionString string
		buildInfo     *debug.BuildInfo
		buildInfoOK   bool
		expected      string
	}{
		{
			name:          "VersionString set",
			versionString: "v1.2.3",
			buildInfo:     &debug.BuildInfo{Main: debug.Module{Version: "v0.0.0"}},
			buildInfoOK:   true,
			expected:      "v1.2.3",
		},
		{
			name:          "BuildInfo available",
			versionString: "",
			buildInfo:     &debug.BuildInfo{Main: debug.Module{Version: "v9.8.7"}},
			buildInfoOK:   true,
			expected:      "v9.8.7",
		},
		{
			name:          "BuildInfo not available",
			versionString: "",
			buildInfo:     nil,
			buildInfoOK:   false,
			expected:      "(unknown)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			mockDebugger := new(mocks.Debugger)
			if test.versionString == "" {
				mockDebugger.On("ReadBuildInfo").Return(test.buildInfo, test.buildInfoOK)
			}

			e := Execute{
				Debugger:      mockDebugger,
				VersionString: test.versionString,
			}
			got := e.Version()
			assert.Equal(t, test.expected, got)
			mockDebugger.AssertExpectations(t)
		})
	}
}

func TestErrorTypeCounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []data.DescriptionError
		want  map[string]int
	}{
		{
			name:  "empty slice",
			input: []data.DescriptionError{},
			want:  map[string]int{},
		},
		{
			name:  "single error with colon",
			input: []data.DescriptionError{{Message: "type-error: something went wrong"}},
			want:  map[string]int{"type-error": 1},
		},
		{
			name:  "single error with space",
			input: []data.DescriptionError{{Message: "field error something went wrong"}},
			want:  map[string]int{"field": 1},
		},
		{
			name: "multiple errors, mixed",
			input: []data.DescriptionError{
				{Message: "type-error: foo"},
				{Message: "type-error: bar"},
				{Message: "field error baz"},
				{Message: "other"},
			},
			want: map[string]int{
				"type-error": 2,
				"field":      1,
				"other":      1,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := errorTypeCounts(test.input)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("errorTypeCounts() = %v, want %v", got, test.want)
			}
		})
	}
}
