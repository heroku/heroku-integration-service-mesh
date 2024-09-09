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
	DataActionTarget string `json:"data_action_target"`
	OrgID            string `json:"org_id"`
	Signature        string `json:"signature"`
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

		// validate request headers
		requestHeader, err := ValidateRequest(r.Header)
		if err != nil {
			slog.Error("Error with validation: " + err.Error())
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Call the endpoint based on type of request
		var finalStatus int
		if requestHeader.IsSalesforceRequest {
			authRequestBody := SalesforceAuthRequestBody{
				OrgDomainUrl: requestHeader.XRequestContext.OrgDomainUrl,
				CoreJWTToken: requestHeader.XRequestContext.Auth,
				OrgID:        requestHeader.XRequestContext.OrgID,
			}

			status, err := callSalesforceAddonAuth(authRequestBody, config.IntegrationUrl, config.InvocationToken, requestHeader.XRequestID)
			if err != nil {
				slog.Error("Error Authorizing Salesforce request from add on: " + err.Error())
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
			status, err := callDataCloudAddonAuth(dataCloudAuthRequestBody, config.IntegrationUrl)
			if err != nil {
				slog.Error("Error Authorizing Datacloud request from add on: " + err.Error())
				http.Error(w, err.Error(), status)
				return
			}
			finalStatus = status

		}

		if finalStatus != http.StatusOK {
			slog.Error("Non-200 Error: " + strconv.Itoa(finalStatus))
			http.Error(w, http.StatusText(finalStatus), finalStatus)
			w.WriteHeader(finalStatus)
			return
		}

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
	url := u + "/datacloud/" + authBody.OrgID + "/data_action_targets/authenticate"
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
