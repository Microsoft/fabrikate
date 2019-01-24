package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstallJSON(t *testing.T) {
	err := Install("../test/fixtures/install")

	assert.Nil(t, err)
}

func TestInstallYAML(t *testing.T) {
	err := Install("../test/fixtures/install-yaml")

	assert.Nil(t, err)
}

func TestInstallWithBeforeHook(t *testing.T) {
	err := Install("../test/fixtures/before-install")

	assert.Nil(t, err)
}
