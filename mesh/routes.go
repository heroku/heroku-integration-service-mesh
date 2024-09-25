package mesh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"main/conf"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Routes struct {
	transport http.RoundTripper
}

type SalesforceAuthRequestBody struct {
	OrgDomainUrl string `json:"org_domain_url"`
	CoreJWTToken string `json:"core_jwt_token"`
	OrgID        string `json:"org_id"`
}

type DataActionTargetAuthRequestBody struct {
	ApiName   string `json:"api_name"`
	OrgID     string `json:"org_id"`
	Signature string `json:"signature"`
	Payload   string `json:"payload"`
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
		startTime := time.Now()

		config := conf.GetConfig()
		should_auth_disable := os.Getenv("HEROKU_INTEGRATION_SERVICE_MESH_AUTH_DISABLE")
		bypass_auth, err := strconv.ParseBool(should_auth_disable)

		// validate request headers
		// Generate requestId; overwritten by x-request-id, if provided
		requestID, requestHeader, err := ValidateRequest(uuid.New().String(), r.Header)
		if err != nil {
			httpStatusCode := http.StatusUnauthorized
			switch err.(type) {
			case *InvalidRequest:
				httpStatusCode = err.(*InvalidRequest).HttpStatusCode()
			default:
			}
			logError(requestID, err.Error())
			http.Error(w, err.Error(), httpStatusCode)
			return
		}

		if !bypass_auth {
			// Call the endpoint based on type of request - Salesforce or Data Action Target
			var finalStatus int

			if requestHeader.IsSalesforceRequest {
				requestID := requestHeader.XRequestID
				logInfo(requestID, "Evaluating Salesforce request for org "+requestHeader.XRequestContext.OrgID+", domain "+
					requestHeader.XRequestContext.OrgDomainUrl)

				authRequestBody := SalesforceAuthRequestBody{
					OrgDomainUrl: requestHeader.XRequestContext.OrgDomainUrl,
					CoreJWTToken: requestHeader.XRequestContext.Auth,
					OrgID:        requestHeader.XRequestContext.OrgID,
				}

				// FIXME: Remove when no longer needed
				logInfo(requestID, "REMOVEME Auth: "+requestHeader.XRequestContext.Auth)

				status, err := callSalesforceAuth(requestID, authRequestBody, config.IntegrationUrl, config.InvocationToken)
				if err != nil {
					logError(requestID, "Error authorizing Salesforce request: "+err.Error())
					http.Error(w, err.Error(), http.StatusUnauthorized)
					return
				}
				finalStatus = status

			} else { // This means that it is a Data Action Target request
				logInfo(requestID, "Evaluating Data Action Target request...")

				// Get data from query params
				queryParams := r.URL.Query()

				// Get the request body from the initial request
				bodyStr, err := io.ReadAll(r.Body)
				if err != nil {
					logError(requestID, "Error parsing body from the request: "+err.Error())
					http.Error(w, err.Error(), http.StatusForbidden)
					return
				}

				// Build Data Action Target authentication request Body
				dataActionTargetAuthRequestBody := DataActionTargetAuthRequestBody{
					ApiName:   queryParams.Get(ApiNameQueryParam),
					OrgID:     queryParams.Get(OrgIdQueryParm),
					Signature: requestHeader.XSignature,
					Payload:   string(bodyStr),
				}

				// Authenticate Data Action Target request
				status, err := callDataTargetActionAuth(requestID, dataActionTargetAuthRequestBody, config.IntegrationUrl)
				if err != nil {
					logError(requestID, "Error authenticating Data Action Target request: "+err.Error())
					http.Error(w, err.Error(), status)
					return
				}

				if status != http.StatusOK {
					status = http.StatusForbidden
				}
				finalStatus = status
			}

			if finalStatus != http.StatusOK {
				if finalStatus == http.StatusForbidden {
					logError(requestID, "Unauthorized request")
				} else {
					logError(requestID, "Failed Integration authentication request: "+strconv.Itoa(finalStatus))

				}
				http.Error(w, http.StatusText(finalStatus), finalStatus)
				w.WriteHeader(finalStatus)
				return
			}
		} else {
			slog.Warn("Bypassed authentication")
		}

		defer timeTrack(requestID, startTime, "Heroku Integration Service Mesh")

		logInfo(requestID, "Authentication successful")

		// Successful auth'd, forward request to API
		forwardApiUrl, err := getForwardUrl(r, config.AppPort)
		logInfo(requestID, "Forwarding request to app API "+forwardApiUrl)
		forwardReq, err := http.NewRequest(r.Method, forwardApiUrl, r.Body)
		if err != nil {
			logError(requestID, "Error assembling request: "+err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		// Apply request headers to forward request
		for header, values := range r.Header {
			for _, value := range values {
				forwardReq.Header.Set(header, value)
			}
		}

		client := &http.Client{}
		resp, err := client.Do(forwardReq)
		if err != nil {
			logError(requestID, "Error forwarding request: "+err.Error())
			http.Error(w, "Error forwarding request "+requestID, http.StatusBadGateway)
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

/**
 * Authenticate Salesforce request.
 *
 * @param requestID
 * @param sfAuthRequestBody
 * @param integrationUrl
 * @param integrationToken
 * @return int status code
 * @return error error if any
 * @return string error message if any
 */
func callSalesforceAuth(requestID string, sfAuthRequestBody SalesforceAuthRequestBody, integrationUrl string, integrationToken string) (int, error) {
	logInfo(requestID, "Authenticating Salesforce request...")
	startTime := time.Now()

	jsonBody, err := json.Marshal(sfAuthRequestBody)
	// Call the Integration service
	// TODO: Remove when no longer needed
	logInfo(requestID, "REMOVEME Calling Integration service "+integrationUrl+" with invocation token "+integrationToken+"...")
	req, err := http.NewRequest(http.MethodPost, integrationUrl+"/invocations/authentication", bytes.NewBuffer(jsonBody))
	if err != nil {
		logError(requestID, fmt.Sprintf("Error assembling Integration Salesforce authentication request: %v", err))
		return http.StatusBadRequest, fmt.Errorf("error assembling Integration Salesforce authentication request %s: %v", requestID, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+integrationToken)
	req.Header.Set("REQUEST_ID", requestID)

	client := &http.Client{}
	resp, err := client.Do(req)
	elapsedTime := time.Since(startTime)
	logInfo(requestID, "Integration Salesforce authentication request took "+elapsedTime.String())
	if err != nil {
		logError(requestID, fmt.Sprintf("Error invoking Integration Salesforce authentication for request: %v", err))
		return http.StatusBadRequest, fmt.Errorf("error invoking Integration Salesforce authentication request %s: %v", requestID, err)
	}

	logInfo(requestID, "Authentication result for Salesforce request: "+strconv.Itoa(resp.StatusCode))

	defer resp.Body.Close()

	return resp.StatusCode, nil

}

/**
 * Authenticate Data Action Target webhook request.
 *
 * @param requestID
 * @param dataActionTargetAuthRequestBody
 * @param integrationUrl
 */
func callDataTargetActionAuth(requestID string, dataActionTargetAuthRequestBody DataActionTargetAuthRequestBody, integrationUrl string) (int, error) {
	logInfo(requestID, "Authenticating Data Action Target '"+dataActionTargetAuthRequestBody.ApiName+"' from org "+
		dataActionTargetAuthRequestBody.OrgID+" with payload length "+strconv.Itoa(len(dataActionTargetAuthRequestBody.Payload))+"...")
	startTime := time.Now()

	jsonBody, err := json.Marshal(dataActionTargetAuthRequestBody)
	datAuthUrl := integrationUrl + "/connections/datacloud/authenticate"
	req, err := http.NewRequest(http.MethodPost, datAuthUrl, bytes.NewBuffer(jsonBody))
	if err != nil {
		logError(requestID, fmt.Sprintf("Error assembling Integration Data Action Target authentication request: %v", err))
		return http.StatusBadRequest, fmt.Errorf("error creating Data Action Target authentication request %s: %v", requestID, err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	elapsedTime := time.Since(startTime)
	logInfo(requestID, "Data Action Target authentication request took "+elapsedTime.String())
	if err != nil {
		logError(requestID, fmt.Sprintf("Error invoking Integration Data Action Target authentication: %v", err))
		return http.StatusBadRequest, fmt.Errorf("error invoking Integration Data Action Target authentication %s: %v", requestID, err)
	}

	defer resp.Body.Close()

	logInfo(requestID, "Authentication result for Data Action Target request: "+strconv.Itoa(resp.StatusCode))

	return resp.StatusCode, nil
}
