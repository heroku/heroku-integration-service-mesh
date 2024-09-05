package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"log"
	"log/slog"
	"main/conf"
	"net/http"
	"os"
	"os/exec"
	"runtime"
)

func main() {

	config := conf.GetConfig()

	app := &cli.App{
		Name:                   "heroku-integration-service-mesh",
		Usage:                  "Service to pass communication between Heroku Integration and Customer App",
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
	logger := slog.With(
		slog.String("app", os.Getenv("HEROKU_APP_NAME")),
	)
	slog.SetDefault(logger)
}
