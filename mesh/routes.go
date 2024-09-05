package mesh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"log/slog"
	"net/http"
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

type AuthRequestBody struct {
	OrgDomainUrl string `json:"org_domain_url"`
	CoreJWTToken string `json:"core_jwt_token"`
	OrgID        string `json:"org_id"`
}

const HEROKU_INTEGRATION_API_URL = "https://heroku-integration-prod-c06ef9c8a54e.herokuapp.com/addons/e271b891-c53a-4240-a510-e0ffb218e416"

func InitializeRoutes(router chi.Router) {
	routes := NewRoutes()
	//router.Post("/", routes.PassThrough())
	router.HandleFunc("/*", routes.Pass())
	router.Post("/salesforce/auth", routes.SalesforceAuth())
}

func NewRoutes() *Routes {
	return &Routes{http.DefaultTransport}
}

func getForwardUrl(r *http.Request) (string, error) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s", scheme, r.Host)
	forwardingUrl := strings.Replace(url, "8070", "3000", 1)
	return forwardingUrl, nil
}
func (routes *Routes) Pass() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		forwardUrl, err := getForwardUrl(r)
		targetUrl := forwardUrl + r.URL.Path
		fmt.Println(targetUrl)

		proxyReq, err := http.NewRequest(r.Method, targetUrl, r.Body)
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

func (routes *Routes) SalesforceAuth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Validate Request
		requestHeader, err := ValidateSalesforceRequest(r.Header, AuthRequest)
		if err != nil {
			slog.Error("Invalid request: %v", err)
		}

		// build the request body
		authRequestBody := AuthRequestBody{
			OrgDomainUrl: requestHeader.XRequestContext.OrgDomainUrl,
			CoreJWTToken: requestHeader.XRequestContext.Auth,
			OrgID:        requestHeader.XRequestContext.OrgID,
		}

		jsonBody, err := json.Marshal(authRequestBody)
		if err != nil {
			slog.Error("Error marshalling auth request body: %v", err)
			http.Error(w, "Error marshalling auth request body", http.StatusBadRequest)

		}

		// call the addon service
		req, err := http.NewRequest(http.MethodPost, HEROKU_INTEGRATION_API_URL+"/invocations/authentication", bytes.NewBuffer(jsonBody))
		if err != nil {
			slog.Error("Error creating auth request: %v", err)
			http.Error(w, "Error creating auth request", http.StatusBadRequest)
		}

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			slog.Error("Error invoking authentication: %v", err)
			http.Error(w, "Error invoking authentication", http.StatusBadRequest)
		}

		defer resp.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
	}
}

//func (routes *Routes) Start() http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		_, err := ValidateRequest(r.Header)
//		if err != nil {
//			slog.Error("Invalid request %v", err)
//			http.Error(w, "Error reading body: "+err.Error(), http.StatusInternalServerError)
//			return
//		}
//
//		// get the command
//		var req StartRequest
//		err = json.NewDecoder(r.Body).Decode(&req)
//		if err != nil {
//			http.Error(w, "Invalid request body", http.StatusBadRequest)
//			return
//		}
//
//		if req.Command == "" {
//			http.Error(w, "Command is required", http.StatusBadRequest)
//			return
//		}
//
//		// split the command string into command arguements
//		cmdArgs := strings.Fields(req.Command)
//		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
//
//		// set up environment variables
//		cmd.Env = os.Environ()
//		for key, value := range req.EnvironmentVariables {
//			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
//		}
//
//		// run the command
//		output, err := cmd.CombinedOutput()
//		if err != nil {
//			http.Error(w, fmt.Sprintf("Error: %v\n%s", err, output), http.StatusInternalServerError)
//			return
//		}
//		w.Header().Set("Content-Type", "application/json")
//		w.WriteHeader(http.StatusOK)
//		w.Write(output)
//	}
//}
