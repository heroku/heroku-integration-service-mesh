package test

import (
	"slices"
	"strings"
	"testing"

	"github.com/heroku/heroku-integration-service-mesh/conf"
)

func Test_GetConfig(t *testing.T) {
	t.Setenv("HEROKU_INTEGRATION_TOKEN", "HEROKU_INTEGRATION_TOKEN")
	t.Setenv("HEROKU_INTEGRATION_API_URL", "HEROKU_INTEGRATION_API_URL")

	config := conf.GetConfig()

	if config.Version == "" {
		t.Error("Should have Version")
	}

	if config.AppPort == "" {
		t.Error("Should have AppPort")
	}

	if config.AppUrl == "" {
		t.Error("Should have AppUrl")
	}

	if config.PublicPort == "" {
		t.Error("Should have PublicPort")
	}
}

func Test_ParseYamlConfig(t *testing.T) {
	yamlConfig, err := conf.ParseYamlConfig(conf.YamlFileName)

	if err != nil {
		t.Error(err)
	}

	if yamlConfig == nil {
		t.Error("Should have YamlConfig")
	}

	bypassRoutes := yamlConfig.Authentication.BypassRoutes
	if len(bypassRoutes) != 2 {
		t.Error("Should have YamlConfig.Authentication.BypassRoutes")
	}

	if !slices.Contains(bypassRoutes, "/bypassThisRoute") {
		t.Error("Should have '/bypassThisRoute' BypassRoutes [" + strings.Join(bypassRoutes, ", ") + "]")
	}

	if !slices.Contains(bypassRoutes, "/bypassThatRoute") {
		t.Error("Should have '/bypassThatRoute' BypassRoutes [" + strings.Join(bypassRoutes, ", ") + "]")
	}
}

func Test_InvalidYamlConfig(t *testing.T) {
	_, err := conf.ParseYamlConfig("heroku-integration-service-mesh-invalid.yaml")

	if err == nil {
		t.Error("Should have invalid YAML error")
	}
}
