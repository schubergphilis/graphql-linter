package rules

import (
	"testing"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
)

type DistanceTestCase struct {
	Name     string
	Source   string
	Target   string
	Expected int
}

func runDistanceTableTest(
	t *testing.T,
	tests []DistanceTestCase,
	testFunc func(string, string) int,
) {
	t.Helper()

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			got := testFunc(test.Source, test.Target)
			if got != test.Expected {
				t.Errorf("got %v, want %v", got, test.Expected)
			}
		})
	}
}

func TestLevenshteinDistance(t *testing.T) {
	t.Parallel()

	tests := []DistanceTestCase{
		{"identical", "kitten", "kitten", 0},
		{"one substitution", "kitten", "sitten", 1},
		{"one insertion", "kitten", "kittens", 1},
		{"one deletion", "kitten", "kittn", 1},
		{"completely different", "abc", "xyz", 3},
		{"empty source", "", "abc", 3},
		{"empty target", "abc", "", 3},
		{"both empty", "", "", 0},
		{"case sensitive", "Kitten", "kitten", 1},
		{"unicode", "café", "coffee", 4},
		{"longer", "intention", "execution", 5},
	}
	runDistanceTableTest(t, tests, LevenshteinDistance)
}

func TestLevenshteinDistance_ExtraCases(t *testing.T) {
	t.Parallel()

	if LevenshteinDistance("", "") != 0 {
		t.Errorf("expected 0 for empty strings")
	}

	if LevenshteinDistance("abc", "") != 3 {
		t.Errorf("expected 3 for abc vs empty")
	}

	if LevenshteinDistance("", "abc") != 3 {
		t.Errorf("expected 3 for empty vs abc")
	}
}

func TestIsSuppressed(t *testing.T) {
	t.Parallel()

	suppressionConfig := createSuppressionConfig([]models.Suppression{
		{File: "foo.graphql", Line: 2, Rule: "rule", Value: "val"},
	})

	got := IsSuppressed("bar/foo.graphql", 2, suppressionConfig, "rule", "val")
	if !got {
		t.Errorf("expected suppression to match")
	}

	got = IsSuppressed("bar/foo.graphql", 3, suppressionConfig, "rule", "val")
	if got {
		t.Errorf("expected suppression not to match")
	}
}

func createSuppressionConfig(suppressions []models.Suppression) *models.LinterConfig {
	return &models.LinterConfig{
		Suppressions: suppressions,
	}
}

func TestMatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		suppression models.Suppression
		filePath    string
		line        int
		rule        string
		value       string
		wantMatch   bool
	}{
		{"all empty", models.Suppression{}, "foo.graphql", 1, "rule", "value", true},
		{
			"file match",
			models.Suppression{File: "foo.graphql"},
			"bar/foo.graphql",
			1,
			"rule",
			"value",
			true,
		},
		{
			"file no match",
			models.Suppression{File: "baz.graphql"},
			"foo.graphql",
			1,
			"rule",
			"value",
			false,
		},
		{"line match", models.Suppression{Line: 2}, "foo.graphql", 2, "rule", "value", true},
		{"line no match", models.Suppression{Line: 3}, "foo.graphql", 2, "rule", "value", false},
		{
			"rule match",
			models.Suppression{Rule: "myrule"},
			"foo.graphql",
			1,
			"myrule",
			"value",
			true,
		},
		{
			"rule no match",
			models.Suppression{Rule: "otherrule"},
			"foo.graphql",
			1,
			"myrule",
			"value",
			false,
		},
		{"value match", models.Suppression{Value: "val"}, "foo.graphql", 1, "rule", "val", true},
		{
			"value no match",
			models.Suppression{Value: "other"},
			"foo.graphql",
			1,
			"rule",
			"val",
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := Matches(test.filePath, test.line, test.rule, test.suppression, test.value)
			if got != test.wantMatch {
				t.Errorf("got %v, want %v", got, test.wantMatch)
			}
		})
	}
}
