package conf

import (
	"fmt"
	"os"
	"sync"

	cli "github.com/urfave/cli/v2"
)

type Config struct {
	PublicPort            string
	PrivatePort           string
	AppPort               string
	HerokuInvocationToken string
	HerokuIntegrationUrl  string
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
	herokuIntegrationToken := os.Getenv("HEROKU_INTEGRATION_INVOCATIONS_TOKEN")
	herokuIntegrationUrl := os.Getenv("HEROKU_INTEGRATION_API_URL")

	if herokuIntegrationUrl == "" || herokuIntegrationToken == "" {
		fmt.Printf("Heroku Integration add-on config vars not set")
		os.Exit(1)
	}

	if appPort == "" {
		appPort = "3000"
	}

	return &Config{
		PublicPort:            "8070",
		PrivatePort:           "8071",
		AppPort:               appPort,
		HerokuInvocationToken: herokuIntegrationToken,
		HerokuIntegrationUrl:  herokuIntegrationUrl,
	}
})

func GetConfig() *Config {
	return defaultConfig()
}
