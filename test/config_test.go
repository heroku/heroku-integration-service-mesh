package test

import (
	"slices"
	"strings"
	"testing"

	"github.com/heroku/heroku-integration-service-mesh/conf"
)

func Test_GetConfigDefaults(t *testing.T) {
	t.Setenv("HEROKU_INTEGRATION_TOKEN", "HEROKU_INTEGRATION_TOKEN")
	t.Setenv("HEROKU_INTEGRATION_API_URL", "HEROKU_INTEGRATION_API_URL")

	config := conf.GetConfig()

	if config.Version == "" {
		t.Error("Should have Version")
	}

	if config.YamlConfig.App.Port != conf.AppPort {
		t.Error("Should have default YamlConfig.App.Port " + conf.AppPort + ", got " + config.YamlConfig.App.Port)
	}

	if config.YamlConfig.App.Host != conf.AppHost {
		t.Error("Should have default YamlConfig.App.Host " + conf.AppHost + ", got " + config.YamlConfig.App.Host)
	}

	if config.PublicPort == "" {
		t.Error("Should have PublicPort")
	}

	if config.YamlConfig.Mesh.HealthCheck.Enable != "true" {
		t.Error("Should have YamlConfig.Mesh.HeathCheck true, got " +
			config.YamlConfig.Mesh.HealthCheck.Enable)
	}

	if config.YamlConfig.Mesh.HealthCheck.Route != conf.HealthCheckRoute {
		t.Error("Should have YamlConfig.Mesh.HeathCheck '" + conf.HealthCheckRoute + "', got " +
			config.YamlConfig.Mesh.HealthCheck.Route)
	}
}

func Test_InitYamlConfig(t *testing.T) {
	yamlConfig, err := conf.InitYamlConfig(conf.YamlFileName)

	if err != nil {
		t.Error(err)
	}

	if yamlConfig == nil {
		t.Error("Should have YamlConfig")
	}

	bypassRoutes := yamlConfig.Mesh.Authentication.BypassRoutes
	if len(bypassRoutes) != 2 {
		t.Error("Should have YamlConfig.Authentication.BypassRoutes")
	}

	if !slices.Contains(bypassRoutes, "/bypassThisRoute") {
		t.Error("Should have '/bypassThisRoute' BypassRoutes [" + strings.Join(bypassRoutes, ", ") + "]")
	}

	if yamlConfig.Mesh.HealthCheck.Enable != "true" {
		t.Error("Should have Healthcheck enabled")
	}

	if yamlConfig.Mesh.HealthCheck.Route == "" {
		t.Error("Should have Healthcheck enabled")
	}
}

func Test_InvalidYamlConfig(t *testing.T) {
	_, err := conf.InitYamlConfig("heroku-integration-service-mesh-invalid.yaml")

	if err == nil {
		t.Error("Should have invalid YAML error")
	}
}

func Test_InitYamlConfigOverrides(t *testing.T) {
	yamlConfig, err := conf.InitYamlConfig("heroku-integration-service-mesh-overrides.yaml")

	if err != nil {
		t.Error(err)
	}

	if yamlConfig.App.Port != "3030" {
		t.Error("Should have YamlConfig.App.Port override " + yamlConfig.App.Port + ", got " + yamlConfig.App.Port)
	}

	if yamlConfig.App.Host != "https://mesh" {
		t.Error("Should have YamlConfig.App.Host override 'https://mesh', got " + yamlConfig.App.Host)
	}

	if yamlConfig.Mesh.HealthCheck.Enable != "false" {
		t.Error("Should have YamlConfig.Mesh.HealthCheck.Enable override false, got " +
			yamlConfig.Mesh.HealthCheck.Enable)
	}
}
