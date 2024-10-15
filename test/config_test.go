package test

import (
	"slices"
	"strings"
	"testing"

	"github.com/heroku/heroku-integration-service-mesh/conf"
)

func Test_ParseYamlConfig(t *testing.T) {
	yamlConfig := conf.ParseYamlConfig()

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
