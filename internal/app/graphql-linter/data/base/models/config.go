package models

type Suppression struct {
	File   string `yaml:"file"`
	Line   int    `yaml:"line"`
	Rule   string `yaml:"rule"`
	Value  string `yaml:"value"`
	Reason string `yaml:"reason"`
}

type Settings struct {
	StrictMode         bool `yaml:"strictMode"`
	ValidateFederation bool `yaml:"validateFederation"`
	CheckDescriptions  bool `yaml:"checkDescriptions"`
}

type LinterConfig struct {
	Suppressions []Suppression `yaml:"suppressions"`
	Settings     Settings      `yaml:"settings"`
}
