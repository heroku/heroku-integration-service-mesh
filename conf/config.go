package conf

import (
	"github.com/urfave/cli/v2"
	"os"
	"sync"
)

type Config struct {
	PublicPort  string
	PrivatePort string
	AppPort     string
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

	if appPort == "" {
		appPort = "3000"
	}

	return &Config{
		PublicPort:  "8070",
		PrivatePort: "8071",
		AppPort:     appPort,
	}
})

func GetConfig() *Config {
	return defaultConfig()
}
