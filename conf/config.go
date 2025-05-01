package conf

import (
	"log"
	"os"
	"strconv"
	"sync"

	cli "github.com/urfave/cli/v2"
	yaml "gopkg.in/yaml.v3"
)

// Heroku Integration authentication API paths
const (
	AppPort                                   = "3000"
	AppHost                                   = "http://127.0.0.1"
	HealthCheckRoute                          = "/healthcheck"
	HerokuIntegrationSalesforceAuthPath       = "/invocations/authentication"
	HerokuIntegrationDataActionTargetAuthPath = "/data_action_targets/authenticate"
	YamlFileName                              = "heroku-integration-service-mesh.yaml"
	AddonAuthUrlFormat                        = "heroku.com/addons/%s/connections/salesforce"
)

type Authentication struct {
	BypassRoutes []string `yaml:"bypassRoutes"`
}

type HealthCheck struct {
	Enable string `yaml:"enable"`
	Route  string `yaml:"route"`
}

type App struct {
	Port string `yaml:"port"`
	Host string `yaml:"host"`
}

type Mesh struct {
	Authentication Authentication `yaml:"authentication"`
	HealthCheck    HealthCheck    `yaml:"healthcheck"`
}

type YamlConfig struct {
	App  App  `yaml:"app"`
	Mesh Mesh `yaml:"mesh"`
}

type Config struct {
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

// TODO: Make customer configurable items configurable in service mesh's YAML file
var defaultConfig = sync.OnceValue(func() *Config {

	// Get env config
	herokuIntegrationToken := os.Getenv("HEROKU_INTEGRATION_TOKEN")
	herokuIntegrationUrl := os.Getenv("HEROKU_INTEGRATION_API_URL")
	shouldBypassAllRoutesConfigVar := os.Getenv("HEROKU_INTEGRATION_SERVICE_MESH_BYPASS_ALL_ROUTES")
	shouldBypassAllRoutes, _ := strconv.ParseBool(shouldBypassAllRoutesConfigVar)

	if herokuIntegrationUrl == "" || herokuIntegrationToken == "" {
		log.Fatal("Heroku Integration add-on config vars not set")
	}

	yamlConfig, err := InitYamlConfig(YamlFileName)
	if err != nil {
		log.Fatalf("Invalid YAML config: %v", err)
	}

	return &Config{
		HerokuInvocationToken:                     herokuIntegrationToken,
		HerokuIntegrationUrl:                      herokuIntegrationUrl,
		HerokuInvocationSalesforceAuthPath:        HerokuIntegrationSalesforceAuthPath,
		HerokuIntegrationDataActionTargetAuthPath: HerokuIntegrationDataActionTargetAuthPath,
		PrivatePort:           "8071",
		PublicPort:            "8070",
		ShouldBypassAllRoutes: shouldBypassAllRoutes,
		Version:               VERSION,
		YamlConfig:            yamlConfig,
	}
})

func InitYamlConfig(yamlFileName string) (*YamlConfig, error) {
	yamlConfig := &YamlConfig{}

	// Parse YAML file, if found
	_, err := os.Stat(yamlFileName)
	if err == nil {
		// Found
		yamlFile, err := os.Open(yamlFileName)
		if err != nil {
			return nil, err
		}
		defer yamlFile.Close()

		decoder := yaml.NewDecoder(yamlFile)
		decoder.KnownFields(true)
		if err := decoder.Decode(&yamlConfig); err != nil {
			return nil, err
		}
	}

	// Apply defaults
	if yamlConfig.App.Port == "" {
		yamlConfig.App.Port = AppPort
	}
	appPort := os.Getenv("APP_PORT")
	if appPort != "" {
		yamlConfig.App.Port = appPort
	}

	if yamlConfig.App.Host == "" {
		yamlConfig.App.Host = AppHost
	}

	if yamlConfig.Mesh.HealthCheck.Enable == "" {
		yamlConfig.Mesh.HealthCheck.Enable = "true"
	}

	if yamlConfig.Mesh.HealthCheck.Route == "" {
		yamlConfig.Mesh.HealthCheck.Route = HealthCheckRoute
	}

	return yamlConfig, nil
}

func GetConfig() *Config {
	return defaultConfig()
}

func GetConfigWithYamlFile() *Config {
	return defaultConfig()
}
