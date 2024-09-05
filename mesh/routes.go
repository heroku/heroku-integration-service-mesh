package mesh

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"log/slog"
	"main/conf"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type Routes struct {
	transport http.RoundTripper
}

type PassThroughResponse struct {
	Header *RequestHeader    `json:"header"`
	Body   map[string]string `json:"body"`
}

type PassResponse struct {
	Header http.Header       `json:"header"`
	Body   map[string]string `json:"body"`
}

type StartRequest struct {
	Command              string            `json:"command"`
	EnvironmentVariables map[string]string `json:"environment_variables"`
}

func InitializeRoutes(router chi.Router) {
	routes := NewRoutes()
	router.Post("/start", routes.Start())
	router.HandleFunc("/*", routes.Pass())
}

func NewRoutes() *Routes {
	return &Routes{http.DefaultTransport}
}

func getForwardUrl(r *http.Request) (string, error) {
	appPort := conf.GetConfig().AppPort

	url := fmt.Sprintf("http://127.0.0.1:%s%s", appPort, r.URL.Path)
	return url, nil
}

func (routes *Routes) Pass() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		forwardUrl, err := getForwardUrl(r)
		proxyReq, err := http.NewRequest(r.Method, forwardUrl, r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		for header, values := range r.Header {
			for _, value := range values {
				proxyReq.Header.Set(header, value)
			}
		}

		client := &http.Client{}
		resp, err := client.Do(proxyReq)
		if err != nil {
			http.Error(w, "Error sending proxy request", http.StatusBadGateway)
		}

		defer resp.Body.Close()

		for header, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(header, value)
			}
		}

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}

func (routes *Routes) Start() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := ValidateRequest(r.Header)
		if err != nil {
			slog.Error("Invalid request %v", err)
			http.Error(w, "Error reading body: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// get the command
		var req StartRequest
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Command == "" {
			http.Error(w, "Command is required", http.StatusBadRequest)
			return
		}

		// split the command string into command arguements
		cmdArgs := strings.Fields(req.Command)
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)

		// set up environment variables
		cmd.Env = os.Environ()
		for key, value := range req.EnvironmentVariables {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}

		// run the command
		output, err := cmd.CombinedOutput()
		if err != nil {
			http.Error(w, fmt.Sprintf("Error: %v\n%s", err, output), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}
}
