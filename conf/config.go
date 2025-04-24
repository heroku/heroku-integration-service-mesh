package conf

import (
	"log"
	"os"
	"strconv"
	"sync"

	cli "github.com/urfave/cli/v2"
	yaml "gopkg.in/yaml.v3"
)

// Heroku AppLink authentication API paths
const (
	AppPort                               = "3000"
	AppHost                               = "http://127.0.0.1"
	HealthCheckRoute                      = "/healthcheck"
	HerokuApplinkSalesforceAuthPath       = "/invocations/authentication"
	HerokuApplinkDataActionTargetAuthPath = "/data_action_targets/authenticate"
	YamlFileName                          = "heroku-applink-service-mesh.yaml"
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
	HerokuInvocationToken                 string
	HerokuApplinkUrl                      string
	HerokuApplinkSalesforceAuthPath       string
	HerokuApplinkDataActionTargetAuthPath string
	PrivatePort                           string
	PublicPort                            string
	ShouldBypassAllRoutes                 bool
	Version                               string
	YamlConfig                            *YamlConfig
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
	herokuApplinkToken := os.Getenv("HEROKU_APPLINK_TOKEN")
	herokuApplinkUrl := os.Getenv("HEROKU_APPLINK_API_URL")
	shouldBypassAllRoutesConfigVar := os.Getenv("HEROKU_APPLINK_SERVICE_MESH_BYPASS_ALL_ROUTES")
	shouldBypassAllRoutes, _ := strconv.ParseBool(shouldBypassAllRoutesConfigVar)

	if herokuApplinkUrl == "" || herokuApplinkToken == "" {
		log.Fatal("Heroku AppLink add-on config vars not set")
	}

	yamlConfig, err := InitYamlConfig(YamlFileName)
	if err != nil {
		log.Fatalf("Invalid YAML config: %v", err)
	}

	return &Config{
		HerokuInvocationToken:                 herokuApplinkToken,
		HerokuApplinkUrl:                      herokuApplinkUrl,
		HerokuApplinkSalesforceAuthPath:       HerokuApplinkSalesforceAuthPath,
		HerokuApplinkDataActionTargetAuthPath: HerokuApplinkDataActionTargetAuthPath,
		PrivatePort:                           "8071",
		PublicPort:                            "8070",
		ShouldBypassAllRoutes:                 shouldBypassAllRoutes,
		Version:                               VERSION,
		YamlConfig:                            yamlConfig,
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
