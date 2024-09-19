package mesh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"log/slog"
	"main/conf"
	"net/http"
	"os"
	"strconv"
)

type Routes struct {
	transport http.RoundTripper
}

type SalesforceAuthRequestBody struct {
	OrgDomainUrl string `json:"org_domain_url"`
	CoreJWTToken string `json:"core_jwt_token"`
	OrgID        string `json:"org_id"`
}

type DataCloudAuthRequestBody struct {
	ApiName   string                 `json:"api_name"`
	OrgID     string                 `json:"org_id"`
	Signature string                 `json:"signature"`
	Payload   map[string]interface{} `json:"payload"`
}

func InitializeRoutes(router chi.Router) {
	routes := NewRoutes()
	router.HandleFunc("/*", routes.Pass())
}

func NewRoutes() *Routes {
	return &Routes{http.DefaultTransport}
}

func getForwardUrl(r *http.Request, appPort string) (string, error) {

	url := fmt.Sprintf("http://127.0.0.1:%s%s", appPort, r.URL.Path)
	return url, nil
}

func (routes *Routes) Pass() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		config := conf.GetConfig()
		should_auth_disable := os.Getenv("HEROKU_INTEGRATION_SERVICE_MESH_AUTH_DISABLE")
		run_auth, err := strconv.ParseBool(should_auth_disable)

		// validate request headers
		requestHeader, err := ValidateRequest(r.Header)
		if err != nil {
			slog.Error("Error with validation: " + err.Error())
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		if !run_auth {
			// Call the endpoint based on type of request
			var finalStatus int
			if requestHeader.IsSalesforceRequest {
				slog.Debug("Building salesforce auth request for add-on")
				authRequestBody := SalesforceAuthRequestBody{
					OrgDomainUrl: requestHeader.XRequestContext.OrgDomainUrl,
					CoreJWTToken: requestHeader.XRequestContext.Auth,
					OrgID:        requestHeader.XRequestContext.OrgID,
				}

				slog.Info("Auth: " + requestHeader.XRequestContext.Auth)

				status, err := callSalesforceAddonAuth(authRequestBody, config.IntegrationUrl, config.InvocationToken, requestHeader.XRequestID)
				if err != nil {
					slog.Error("Error Authorizing Salesforce request from add on: " + err.Error())
					http.Error(w, err.Error(), http.StatusUnauthorized)
					return
				}
				slog.Info("Response has been received from add-on about Salesforce auth request")
				finalStatus = status

			} else { // This means that it is a data cloud request

				// Get data from query params
				queryParams := r.URL.Query()

				// Get the request body from the initial request
				var bodyData map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&bodyData)
				if err != nil {
					slog.Error("Error parsing body from the request: " + err.Error())
					http.Error(w, err.Error(), http.StatusForbidden)
					return
				}

				// Build DataCloudAuth Request Body
				dataCloudAuthRequestBody := DataCloudAuthRequestBody{
					ApiName:   queryParams.Get(ApiNameQueryParam),
					OrgID:     queryParams.Get(OrgIdQueryParm),
					Signature: requestHeader.XSignature,
					Payload:   bodyData,
				}

				// call the addon
				status, err := callDataCloudAddonAuth(dataCloudAuthRequestBody, config.IntegrationUrl)
				if err != nil {
					slog.Error("Error Authorizing Datacloud request from add on: " + err.Error())
					http.Error(w, err.Error(), status)
					return
				}
				slog.Info("Datacloud request has been received from add-on about Datacloud request")
				if status != http.StatusOK {
					status = http.StatusForbidden
				}
				finalStatus = status

			}

			if finalStatus != http.StatusOK {
				slog.Error("Non-200 response from add-on: " + strconv.Itoa(finalStatus))
				http.Error(w, http.StatusText(finalStatus), finalStatus)
				w.WriteHeader(finalStatus)
				return
			}

		}
		slog.Info("Forwarding the request")
		forwardUrl, err := getForwardUrl(r, config.AppPort)
		proxyReq, err := http.NewRequest(r.Method, forwardUrl, r.Body)
		if err != nil {
			slog.Error("Error creating request: " + err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		// adding the same headers for the request
		for header, values := range r.Header {
			for _, value := range values {
				proxyReq.Header.Set(header, value)
			}
		}

		client := &http.Client{}
		resp, err := client.Do(proxyReq)
		if err != nil {
			slog.Error("Error sending proxy request: " + err.Error())
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

func callSalesforceAddonAuth(authBody SalesforceAuthRequestBody, url, token, requestID string) (int, error) {

	jsonBody, err := json.Marshal(authBody)
	// call the addon service
	slog.Debug("Calling Salesforce addon: " + url)
	slog.Debug("Using Token: " + token)
	req, err := http.NewRequest(http.MethodPost, url+"/invocations/authentication", bytes.NewBuffer(jsonBody))
	if err != nil {
		slog.Error("Error creating auth request: %v", err)
		return http.StatusBadRequest, fmt.Errorf("error creating auth request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("REQUEST_ID", requestID)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error invoking authentication: %v", err)
		return http.StatusBadRequest, fmt.Errorf("error invoking authentication %v", err)
	}

	fmt.Printf("Request Went Through: %s", resp.Status)

	defer resp.Body.Close()

	return resp.StatusCode, nil

}

func callDataCloudAddonAuth(authBody DataCloudAuthRequestBody, u string) (int, error) {

	jsonBody, err := json.Marshal(authBody)
	url := u + "connections/datacloud/authenticate"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		slog.Error("Error creating auth request: %v", err)
		return http.StatusBadRequest, fmt.Errorf("error creating auth request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error invoking authentication: %v", err)
		return http.StatusBadRequest, fmt.Errorf("error invoking authentication %v", err)
	}

	defer resp.Body.Close()

	return resp.StatusCode, nil

}
