package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	ContentTypes map[string]MappingConfig     `yaml:"content-types"`
	HealthChecks map[string]HealthCheckConfig `yaml:"end-point-health-checks"`
}

type MappingConfig struct {
	Mapper   string `yaml:"mapper"`
	Endpoint string `yaml:"end-point"`
}

type HealthCheckConfig struct {
	ID               string `yaml:"id"`
	BusinessImpact   string `yaml:"business-impact"`
	Name             string `yaml:"name"`
	PanicGuide       string `yaml:"panic-guide"`
	Severity         uint8  `yaml:"severity"`
	TechnicalSummary string `yaml:"technical-summary"`
	CheckerName      string `yaml:"checker-name"`
}

func ReadConfig(yml string) (*Config, error) {
	by, err := ioutil.ReadFile(yml)
	if err != nil {
		return nil, err
	}

	cfg := &Config{make(map[string]MappingConfig), make(map[string]HealthCheckConfig)}
	err = yaml.Unmarshal(by, cfg)
	if err != nil {
		cfg = nil
	}

	return cfg, err
}
