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
	DataActionTarget string `json:"data_action_target"`
	OrgID            string `json:"org_id"`
	Signature        string `json:"signature"`
}

const HEROKU_INTEGRATION_API_URL = "https://heroku-integration-prod-c06ef9c8a54e.herokuapp.com/addons/e271b891-c53a-4240-a510-e0ffb218e416"

func InitializeRoutes(router chi.Router) {
	routes := NewRoutes()
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

		// validate request headers
		requestHeader, err := ValidateRequest(r.Header)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		// Call the endpoint based on type of request
		var finalStatus int
		if requestHeader.IsSalesforceRequest {
			authRequestBody := SalesforceAuthRequestBody{
				OrgDomainUrl: requestHeader.XRequestContext.OrgDomainUrl,
				CoreJWTToken: requestHeader.XRequestContext.Auth,
				OrgID:        requestHeader.XRequestContext.OrgID,
			}

			status, err := callSalesforceAddonAuth(authRequestBody)
			if err != nil {
				http.Error(w, err.Error(), status)
				return
			}
			finalStatus = status

		} else {

			// Get data from query params
			queryParams := r.URL.Query()

			// Build DataCloudAuth Request Body
			dataCloudAuthRequestBody := DataCloudAuthRequestBody{
				DataActionTarget: queryParams.Get(DataActionTargetQueryParm),
				OrgID:            queryParams.Get(OrgIdQueryParm),
				Signature:        requestHeader.XSignature,
			}

			// call the addon
			status, err := callDataCloudAddonAuth(dataCloudAuthRequestBody)
			if err != nil {
				http.Error(w, err.Error(), status)
				return
			}
			finalStatus = status

		}

		if finalStatus != http.StatusOK {
			http.Error(w, http.StatusText(finalStatus), finalStatus)
			w.WriteHeader(finalStatus)
			return
		}

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

func callSalesforceAddonAuth(authBody SalesforceAuthRequestBody) (int, error) {

	jsonBody, err := json.Marshal(authBody)
	// call the addon service
	req, err := http.NewRequest(http.MethodPost, HEROKU_INTEGRATION_API_URL+"/invocations/authentication", bytes.NewBuffer(jsonBody))
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

func callDataCloudAddonAuth(authBody DataCloudAuthRequestBody) (int, error) {

	jsonBody, err := json.Marshal(authBody)
	// TODO:: Check on productionOrgDC
	url := HEROKU_INTEGRATION_API_URL + "/datacloud/productionOrgDC/data_action_targets/authenticate"
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
