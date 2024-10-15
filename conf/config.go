package conf

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	cli "github.com/urfave/cli/v2"
	yaml "gopkg.in/yaml.v2"
)

// Heroku Integration authentication API paths
const (
	HerokuIntegrationSalesforceAuthPath       = "/invocations/authentication"
	HerokuIntegrationDataActionTargetAuthPath = "/data_action_targets/authenticate"
)

type YamlConfig struct {
	Authentication struct {
		BypassRoutes []string `yaml:"bypassRoutes"`
	}
}

type Config struct {
	AppPort                                   string
	AppUrl                                    string
	HerokuInvocationToken                     string
	HerokuIntegrationUrl                      string
	HerokuInvocationSalesforceAuthPath        string
	HerokuIntegrationDataActionTargetAuthPath string
	PrivatePort                               string
	PublicPort                                string
	ShouldBypassAllRoutes                     bool
	Version                                   string
	YamlConfig                                *YamlConfig
}

func (c *Config) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "port",
			Aliases:     []string{"p"},
			Usage:       "HTTP Port for routes available on the public internet",
			EnvVars:     []string{"PORT"},
			Value:       c.PublicPort,
			Destination: &c.PublicPort,
		},
	}
}

var defaultConfig = sync.OnceValue(func() *Config {

	appPort := os.Getenv("APP_PORT")
	appUrl := os.Getenv("APP_URL")
	herokuIntegrationToken := os.Getenv("HEROKU_INTEGRATION_TOKEN")
	herokuIntegrationUrl := os.Getenv("HEROKU_INTEGRATION_API_URL")
	shouldBypassAllRoutesConfigVar := os.Getenv("HEROKU_INTEGRATION_SERVICE_MESH_BYPASS_ALL_ROUTES")
	shouldBypassAllRoutes, _ := strconv.ParseBool(shouldBypassAllRoutesConfigVar)

	if herokuIntegrationUrl == "" || herokuIntegrationToken == "" {
		fmt.Printf("Heroku Integration add-on config vars not set")
		os.Exit(1)
	}

	if appPort == "" {
		appPort = "3000"
	}

	if appUrl == "" {
		appUrl = "http://127.0.0.1"
	}

	yamlConfigInst := ParseYamlConfig()

	return &Config{
		AppPort:                            appPort,
		AppUrl:                             appUrl,
		HerokuInvocationToken:              herokuIntegrationToken,
		HerokuIntegrationUrl:               herokuIntegrationUrl,
		HerokuInvocationSalesforceAuthPath: HerokuIntegrationSalesforceAuthPath,
		HerokuIntegrationDataActionTargetAuthPath: HerokuIntegrationDataActionTargetAuthPath,
		PrivatePort:           "8071",
		PublicPort:            "8070",
		ShouldBypassAllRoutes: shouldBypassAllRoutes,
		Version:               VERSION,
		YamlConfig:            yamlConfigInst,
	}
})

func ParseYamlConfig() *YamlConfig {
	yamlConfig := &YamlConfig{}

	if _, err := os.Stat("heroku-integration-service-mesh.yaml"); err == nil {
		f, err := os.Open("heroku-integration-service-mesh.yaml")
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		decoder := yaml.NewDecoder(f)
		if err := decoder.Decode(&yamlConfig); err != nil {
			log.Fatal(err)
		}
	}

	return yamlConfig
}

func GetConfig() *Config {
	return defaultConfig()
}
