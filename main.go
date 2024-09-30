package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"runtime"

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
		Version:                fmt.Sprintf("%s [os: %s arch: %s]", VERSION, runtime.GOOS, runtime.GOARCH),
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

	slog.Info("environment",
		slog.String("go_version:", runtime.Version()),
		slog.String("os", runtime.GOOS),
		slog.String("arch", runtime.GOARCH),
		slog.String("http_port", port),
		slog.String("version", VERSION),
		slog.String("environment", env),
		slog.String("app_port", conf.GetConfig().AppPort),
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
	logger := slog.New(slogenv.NewHandler(slog.NewTextHandler(os.Stderr, nil)).WithAttrs([]slog.Attr{
		slog.String("app", os.Getenv("HEROKU_APP_NAME")),
		slog.String("source", "heroku-integration-service-mesh"),
	}))
	slog.SetDefault(logger)
}
