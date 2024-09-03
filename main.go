package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"log"
	"log/slog"
	"net/http"
	"os"
	"runtime"
)

type Config struct {
	PublicPort  string
	PrivatePort string
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

func main() {

	config := &Config{
		PublicPort:  "8070",
		PrivatePort: "8071",
	}

	app := &cli.App{
		Name:                   "heroku-integration-service-mesh",
		Usage:                  "Service to pass communication between Heroku Integration and Customer App",
		UseShortOptionHandling: true,
		Version:                fmt.Sprintf("%s [os: %s arch: %s]", VERSION, runtime.GOOS, runtime.GOARCH),
		Action:                 startServer,
		Flags:                  config.Flags(),
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}

func startServer(c *cli.Context) error {
	setEnvDefault("HEROKU_APP_NAME", "local")
	setEnvDefault("ENVIRONMENT", "local")
	setDefaultLogger()

	env := os.Getenv("ENVIRONMENT")

	port := c.String("port")

	slog.Info("environment",
		slog.String("go_version:", runtime.Version()),
		slog.String("os", runtime.GOOS),
		slog.String("arch", runtime.GOARCH),
		slog.String("http_port", port),
		slog.String("version", VERSION),
		slog.String("environment", env),
	)

	router := NewRouter()
	slog.Info("router running", slog.String("port", port))
	return http.ListenAndServe(":"+port, router)
}

func setEnvDefault(key, fallback string) {
	if _, ok := os.LookupEnv(key); !ok {
		os.Setenv(key, fallback)
	}
}

func setDefaultLogger() {
	logger := slog.With(
		slog.String("app", os.Getenv("HEROKU_APP_NAME")),
	)
	slog.SetDefault(logger)
}
