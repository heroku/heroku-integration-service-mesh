package mesh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/heroku/heroku-integration-service-mesh/conf"
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
	router.HandleFunc("/*", routes.ServiceMesh())
}

func NewRoutes() *Routes {
	return &Routes{http.DefaultTransport}
}

func getForwardUrl(r *http.Request, appPort string) (string, error) {
	url := fmt.Sprintf("http://127.0.0.1:%s%s", appPort, r.URL.RequestURI())
	return url, nil
}

// ServiceMesh intercepts Heroku Integration app API requests validating and authenticating
// each request based on type - Salesforce or Data Action Target.
//
// For validation rules, see ValidateRequest.
//
// Requests are authenticated with the app's associated Heroku Integration add-on
// resource.
//
// If validation and authentication are successful, the mesh forwards the request
// to the target app API.
func (routes *Routes) ServiceMesh() http.HandlerFunc {
	return func(incomingRespWriter http.ResponseWriter, incomingReq *http.Request) {
		startTime := time.Now()
		config := conf.GetConfig()
		apiPath := incomingReq.URL.Path

		// Bypass circuits
		should_auth_disable := os.Getenv("HEROKU_INTEGRATION_SERVICE_MESH_AUTH_DISABLE")
		bypass_auth, err := strconv.ParseBool(should_auth_disable)

		// Get requestID from header; if not found, generate and set
		requestID := incomingReq.Header.Get(HdrNameRequestID)
		if requestID == "" {
			requestID = uuid.New().String()
			incomingRespWriter.Header().Set(HdrNameRequestID, requestID)
			logWarn(requestID, HdrNameRequestID+" not found! Generated and set "+requestID)
		}

		// Log request
		logInfo(requestID, "Processing request to "+apiPath+"...")

		// Validate request headers
		requestHeader, err := ValidateRequest(requestID, incomingReq.Header)
		if err != nil {
			httpStatusCode := http.StatusUnauthorized
			switch err.(type) {
			case *InvalidRequest:
				httpStatusCode = err.(*InvalidRequest).HttpStatusCode()
			default:
			}
			logError(requestID, err.Error())
			http.Error(incomingRespWriter, err.Error(), httpStatusCode)
			return
		}

		// Get the request body from the incoming request
		incomingReqBody, err := io.ReadAll(incomingReq.Body)
		if err != nil {
			logError(requestID, "Unable to parse incoming request body: "+err.Error())
			http.Error(incomingRespWriter, err.Error(), http.StatusBadRequest)
			return
		}

		var orgId string
		var authResponseStatus int
		var authResponseBody string
		if !bypass_auth {
			// Call Integration endpoint based on type of request - Salesforce or Data Action Target
			var unauthorizedMsg string

			if requestHeader.IsSalesforceRequest {
				logInfo(requestID, "Found Salesforce request")

				orgId = requestHeader.XRequestContext.OrgID
				unauthorizedMsg = "Org " + orgId + " not found or not connected to app"
				authRequestBody := SalesforceAuthRequestBody{
					OrgDomainUrl: requestHeader.XRequestContext.OrgDomainUrl,
					CoreJWTToken: requestHeader.XRequestContext.Auth,
					OrgID:        requestHeader.XRequestContext.OrgID,
				}

				// FIXME: Remove when no longer needed
				logDebug(requestID, "!! REMOVEME !! Auth: "+requestHeader.XRequestContext.Auth)

				authResponseStatus, authResponseBody, err = invokeSalesforceAuth(requestID, config, authRequestBody)
				if err != nil {
					logError(requestID, "Unable to authenticate Salesforce request: "+err.Error())
					http.Error(incomingRespWriter, err.Error(), authResponseStatus)
					return
				}

			} else { // This means that it is a Data Action Target request
				logInfo(requestID, "Found Data Action Target request")

				// Get data from query params
				queryParams := incomingReq.URL.Query()

				// Build Data Action Target authentication request Body
				orgId = queryParams.Get(OrgIdQueryParm)
				dataActionTargetAuthRequestBody := DataActionTargetAuthRequestBody{
					ApiName:   queryParams.Get(ApiNameQueryParam),
					OrgID:     orgId,
					Signature: requestHeader.XSignature,
					Payload:   string(incomingReqBody),
				}
				unauthorizedMsg = "Org " + orgId + " not found or not connected to app and/or Data Action Target '" +
					dataActionTargetAuthRequestBody.ApiName + "' signed key not found or is invalid"

				// Authenticate Data Action Target request
				authResponseStatus, authResponseBody, err = invokeDataTargetActionAuth(requestID, config, dataActionTargetAuthRequestBody)
				if err != nil {
					logError(requestID, "Unable to authenticate Data Action Target request: "+err.Error())
					http.Error(incomingRespWriter, err.Error(), authResponseStatus)
					return
				}
			}

			// Handle unauthorized or unexpected failed auth requests
			if authResponseStatus != http.StatusOK {
				if authResponseStatus == http.StatusUnauthorized || authResponseStatus == http.StatusForbidden {
					logWarn(requestID, "Unauthorized request! "+unauthorizedMsg)
					// Unauthenticated requests that appear to be valid are 403 Forbidden
					http.Error(incomingRespWriter, http.StatusText(http.StatusForbidden), http.StatusForbidden)
					incomingRespWriter.WriteHeader(authResponseStatus)
				} else {
					// Unexpected error
					logError(requestID, "Unable to authenticate request: statusCode "+strconv.Itoa(authResponseStatus)+", body '"+authResponseBody+"'")
					http.Error(incomingRespWriter, authResponseBody, authResponseStatus)
					incomingRespWriter.WriteHeader(authResponseStatus)
				}

				// Do NOT forward
				return
			}
		} else {
			logWarn(requestID, "Bypassed authentication")
		}

		// Successful authentication!
		logInfo(requestID, "Authenticated request!")

		// Forward request to target API
		forwardApiUrl, err := getForwardUrl(incomingReq, config.AppPort)
		logInfo(requestID, "Forwarding request to "+forwardApiUrl)
		forwardReq, err := http.NewRequest(incomingReq.Method, forwardApiUrl, bytes.NewReader(incomingReqBody))
		if err != nil {
			logError(requestID, "Unable to forward request: "+err.Error())
			http.Error(incomingRespWriter, err.Error(), http.StatusInternalServerError)
		}

		// Apply request headers to forward request
		for header, values := range incomingReq.Header {
			for _, value := range values {
				forwardReq.Header.Set(header, value)
			}
		}

		// Forward request
		client := &http.Client{}
		forwardResp, err := client.Do(forwardReq)
		if err != nil {
			logError(requestID, "Unable to forward request: "+err.Error())
			http.Error(incomingRespWriter, "Unable to forward request "+requestID, http.StatusBadGateway)
		}

		// Copy incoming headers to forward request
		for header, values := range forwardResp.Header {
			for _, value := range values {
				incomingRespWriter.Header().Add(header, value)
			}
		}

		timeTrack(requestID, startTime, "Heroku Integration Service Mesh")

		// Copy forward request to incoming response
		incomingRespWriter.WriteHeader(forwardResp.StatusCode)
		io.Copy(incomingRespWriter, forwardResp.Body)
		defer forwardResp.Body.Close()
	}
}

// Authenticate Salesforce request
func invokeSalesforceAuth(requestID string, config *conf.Config, sfAuthRequestBody SalesforceAuthRequestBody) (int, string, error) {
	logInfo(requestID, "Authenticating Salesforce request for org "+sfAuthRequestBody.OrgID+", domain "+
		sfAuthRequestBody.OrgDomainUrl+"...")

	operation := "Salesforce authentication"
	jsonBody, err := json.Marshal(sfAuthRequestBody)
	statusCode, body, err := invokeHerokuIntegrationService(requestID, config, operation, config.HerokuInvocationSalesforceAuthPath,
		http.MethodPost, jsonBody)

	return statusCode, body, err
}

// Authenticate Data Action Target webhook request
func invokeDataTargetActionAuth(requestID string, config *conf.Config, dataActionTargetAuthRequestBody DataActionTargetAuthRequestBody) (int, string, error) {
	logInfo(requestID, "Authenticating Data Action Target '"+dataActionTargetAuthRequestBody.ApiName+"' request from org "+
		dataActionTargetAuthRequestBody.OrgID+" with payload length "+strconv.Itoa(len(dataActionTargetAuthRequestBody.Payload))+"...")

	operation := "Data Action Target authentication"
	jsonBody, err := json.Marshal(dataActionTargetAuthRequestBody)
	statusCode, body, err := invokeHerokuIntegrationService(requestID, config, operation, config.HerokuIntegrationDataActionTargetAuthPath,
		http.MethodPost, jsonBody)

	return statusCode, body, err
}

// Invoke given Heroku Integration API with given request JSON body.
func invokeHerokuIntegrationService(requestID string, config *conf.Config, operation string, apiPath string, httpMethod string, jsonBody []byte) (int, string, error) {
	defer timeTrack(requestID, time.Now(), operation)

	statusCode := http.StatusInternalServerError
	body := ""

	integrationApiUrl := config.HerokuIntegrationUrl + apiPath
	req, err := http.NewRequest(httpMethod, integrationApiUrl, bytes.NewBuffer(jsonBody))
	if err != nil {
		logError(requestID, fmt.Sprintf("Unable to assemble %s request: %v", operation, err))
		return statusCode, body, fmt.Errorf("unable to assemble %s request %s: %v", operation, requestID, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.HerokuInvocationToken)
	req.Header.Set("REQUEST_ID", requestID)

	// TODO: Remove when no longer needed
	logDebug(requestID, fmt.Sprintf("!! REMOVEME !! Calling Heroku Integration API %s [%s] with body %s",
		integrationApiUrl, config.HerokuInvocationToken, jsonBody))

	// Invoke
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logError(requestID, fmt.Sprintf("Unable to invoke %s: %v", operation, err))
		return statusCode, body, fmt.Errorf("unable to invoke %s request %s: %v", operation, requestID, err)
	}

	defer resp.Body.Close()

	// Capture statusCode and body
	statusCode = resp.StatusCode
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logError(requestID, err.Error())
	} else {
		body = string(bodyBytes)
	}

	logInfo(requestID, "Response for "+operation+" request ("+apiPath+"): statusCode "+strconv.Itoa(statusCode)+", body '"+body+"'")

	return statusCode, body, nil
}
