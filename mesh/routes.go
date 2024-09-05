package mesh

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"log/slog"
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
	//router.Post("/", routes.PassThrough())
	router.Post("/start", routes.Start())
	router.HandleFunc("/*", routes.Pass())
}

func NewRoutes() *Routes {
	return &Routes{http.DefaultTransport}
}

func (routes *Routes) Pass() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error("Error reading body %v", err)
			http.Error(w, "Error reading body: "+err.Error(), http.StatusInternalServerError)
			return
		}

		defer r.Body.Close()

		// transform body into a json format
		var data map[string]string
		if len(body) > 0 {
			err = json.Unmarshal(body, &data)
			if err != nil {
				http.Error(w, "Error decoding JSON", http.StatusBadRequest)
				return
			}
		}

		response := &PassResponse{
			Header: r.Header,
			Body:   data,
		}

		//convert entire response to JSON
		resp, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Error creating JSON response", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(resp)
	}
}

func (routes *Routes) PassThrough() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Validate the request
		requestHeader, err := ValidateRequest(r.Header)
		if err != nil {
			slog.Error("Invalid request %v", err)
			http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error("Error reading body %v", err)
			http.Error(w, "Error reading body: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// transform body into a json format
		var data map[string]string
		err = json.Unmarshal(body, &data)
		if err != nil {
			http.Error(w, "Error decoding JSON", http.StatusBadRequest)
			return
		}

		defer r.Body.Close()

		response := &PassThroughResponse{
			Header: requestHeader,
			Body:   data,
		}

		//convert entire response to JSON
		resp, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Error creating JSON response", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(resp)
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
