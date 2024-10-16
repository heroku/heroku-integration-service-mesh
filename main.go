package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	slogenv "github.com/cbrewster/slog-env"
	"github.com/heroku/heroku-integration-service-mesh/conf"
	cli "github.com/urfave/cli/v2"
)

func main() {
	config := conf.GetConfig()

	app := &cli.App{
		Name:                   "heroku-integration-service-mesh",
		Usage:                  "Service that handles validation and authentication between clients and Heroku apps",
		UseShortOptionHandling: true,
		Version:                fmt.Sprintf("%s [os: %s, arch: %s]", config.Version, runtime.GOOS, runtime.GOARCH),
		Action:                 startServer,
		Flags:                  config.Flags(),
	}

	go func() {
		if err := app.Run(os.Args); err != nil {
			log.Fatal(err)
		}
	}()

	if len(os.Args) > 1 {
		// get the commands
		cmd := exec.Command(os.Args[1], os.Args[2:]...)

		// Set the command's stdout and stderr to os.Stdout and os.Stderr
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Start()

		// execute the command
		err = cmd.Wait()
		if err != nil {
			fmt.Printf("Error executing command: %v\n", err)
			os.Exit(1)
		}
	}

}

func startServer(c *cli.Context) error {
	setEnvDefault("HEROKU_APP_NAME", "local")
	setEnvDefault("ENVIRONMENT", "local")
	setDefaultLogger()

	env := os.Getenv("ENVIRONMENT")

	port := c.String("port")

	config := conf.GetConfig()

	slog.Info("environment",
		slog.String("go_version:", runtime.Version()),
		slog.String("os", runtime.GOOS),
		slog.String("arch", runtime.GOARCH),
		slog.String("http_port", port),
		slog.String("version", config.Version),
		slog.String("environment", env),
		slog.String("app_host", config.YamlConfig.App.Host),
		slog.String("app_port", config.YamlConfig.App.Port),
	)

	router := NewRouter()
	slog.Info("Heroku Integration Service Mesh is up!", slog.String("port", port))

	if len(config.YamlConfig.Mesh.Authentication.BypassRoutes) > 0 {
		slog.Warn("Authentication bypass routes: " + strings.Join(config.YamlConfig.Mesh.Authentication.BypassRoutes, ", "))
	}

	return http.ListenAndServe(":"+port, router)
}

func setEnvDefault(key, fallback string) {
	if _, ok := os.LookupEnv(key); !ok {
		os.Setenv(key, fallback)
	}
}

func setDefaultLogger() {
	logger := slog.New(slogenv.NewHandler(slog.NewTextHandler(os.Stderr, nil)).WithAttrs([]slog.Attr{
		slog.String("app", os.Getenv("HEROKU_APP_NAME")),
		slog.String("source", "heroku-integration-service-mesh"),
	}))
	slog.SetDefault(logger)
}
