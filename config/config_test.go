package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadConfig(t *testing.T) {
	cfg, err := ReadConfig("../config.yml")

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestReadConfigNotFound(t *testing.T) {
	cfg, err := ReadConfig("./no-such-file.yml")

	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestReadConfigNotValid(t *testing.T) {
	cfg, err := ReadConfig("../Dockerfile")

	assert.Error(t, err)
	assert.Nil(t, cfg)
}
