package mesh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
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

func GetForwardUrl(appUrl string, appPort string, forwardApiPath *http.Request) (string, error) {
	url := fmt.Sprintf("%s:%s%s", appUrl, appPort, forwardApiPath.URL.RequestURI())
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

		if apiPath == InfoRoute {
			// TODO: What addt'l info is useful?
			info := fmt.Sprintf("%s", config.Version)
			LogInfo("n/a", info)
			_, err := fmt.Fprintf(incomingRespWriter, info)
			if err != nil {
				http.Error(incomingRespWriter, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		// Get requestID from header; if not found, generate and set
		requestID := incomingReq.Header.Get(HdrNameRequestID)
		if requestID == "" {
			requestID = uuid.New().String()
			incomingRespWriter.Header().Set(HdrNameRequestID, requestID)
			LogWarn(requestID, HdrNameRequestID+" not found! Generated and set "+requestID)
		}

		// Log request
		LogInfo(requestID, "Processing request to "+apiPath+"...")

		// Bypass ALL routes or incoming route?
		shouldBypassValidationAuthentication := config.ShouldBypassAllRoutes ||
			(len(config.YamlConfig.Authentication.BypassRoutes) > 0 &&
				slices.Contains(config.YamlConfig.Authentication.BypassRoutes, apiPath))
		if shouldBypassValidationAuthentication {
			LogWarn(requestID, "Bypassing authentication and validation for route "+apiPath)
		}

		// Get the request body from the incoming request
		incomingReqBody, err := io.ReadAll(incomingReq.Body)
		if err != nil {
			LogError(requestID, "Failed to parse incoming request body: "+err.Error())
			http.Error(incomingRespWriter, err.Error(), http.StatusBadRequest)
			return
		}

		// Validate and authenticate request, maybe
		if !shouldBypassValidationAuthentication {
			// Validate request headers
			isValid, requestHeader := ValidateRequestHandler(requestID, incomingRespWriter, incomingReq)
			if !isValid {
				// Log time took to evaluate request
				TimeTrack(requestID, startTime, "Heroku Integration Service Mesh")

				// Not valid, do not forward request
				return
			}

			// Authenticate request
			isAuthenticated := AuthenticateRequest(requestID, config, requestHeader, incomingRespWriter, incomingReq, incomingReqBody)
			if !isAuthenticated {
				// Log time took to evaluate request
				TimeTrack(requestID, startTime, "Heroku Integration Service Mesh")

				// Not authorized, do not forward request
				return
			}
		}

		// Forward request to target API; send response to incoming request
		forwardApiUrl, err := GetForwardUrl(config.AppUrl, config.AppPort, incomingReq)
		ForwardRequestReplyToIncomingRequest(startTime, requestID, forwardApiUrl, incomingRespWriter, incomingReq, incomingReqBody)
	}
}

// ValidateRequestHandler Validate request headers
func ValidateRequestHandler(requestID string, incomingRespWriter http.ResponseWriter, incomingReq *http.Request) (bool, *RequestHeader) {
	requestHeader, err := ValidateRequest(requestID, incomingReq.Header)
	if err != nil {
		httpStatusCode := http.StatusUnauthorized
		switch err.(type) {
		case *InvalidRequest:
			httpStatusCode = err.(*InvalidRequest).HttpStatusCode()
		default:
		}
		LogError(requestID, err.Error())
		http.Error(incomingRespWriter, err.Error(), httpStatusCode)
		return false, nil
	}

	return true, requestHeader
}

// AuthenticateRequest Authenticate request based on request type - Salesforce or Data Action Target
func AuthenticateRequest(
	requestID string,
	config *conf.Config,
	requestHeader *RequestHeader,
	incomingRespWriter http.ResponseWriter,
	incomingReq *http.Request,
	incomingReqBody []byte) bool {

	var orgId string
	var authResponseStatus int
	var authResponseBody string
	var unauthorizedMsg string
	var err error

	if requestHeader.IsSalesforceRequest {
		LogInfo(requestID, "Found Salesforce request")

		orgId = requestHeader.XRequestContext.OrgID
		unauthorizedMsg = "Org " + orgId + " not found or not connected to app"
		authRequestBody := SalesforceAuthRequestBody{
			OrgDomainUrl: requestHeader.XRequestContext.OrgDomainUrl,
			CoreJWTToken: requestHeader.XRequestContext.Auth,
			OrgID:        requestHeader.XRequestContext.OrgID,
		}

		// FIXME: Remove when no longer needed
		LogDebug(requestID, "!! REMOVEME !! Auth: "+requestHeader.XRequestContext.Auth)

		authResponseStatus, authResponseBody, err = InvokeSalesforceAuth(requestID, config, authRequestBody)
		if err != nil {
			LogError(requestID, "Failed to authenticate Salesforce request: "+err.Error())
			http.Error(incomingRespWriter, err.Error(), authResponseStatus)
			return false
		}
	} else {
		// Found Data Action Target request
		LogInfo(requestID, "Found Data Action Target request")

		// Get data from query params
		queryParams := incomingReq.URL.Query()

		// Build Data Action Target authentication request Body
		orgId = queryParams.Get(OrgIdQueryParam)
		dataActionTargetAuthRequestBody := DataActionTargetAuthRequestBody{
			ApiName:   queryParams.Get(ApiNameQueryParam),
			OrgID:     orgId,
			Signature: requestHeader.XSignature,
			Payload:   string(incomingReqBody),
		}
		unauthorizedMsg = "Org " + orgId + " not found or not connected to app and/or Data Action Target '" +
			dataActionTargetAuthRequestBody.ApiName + "' signed key not found or is invalid"

		// Authenticate Data Action Target request
		authResponseStatus, authResponseBody, err = InvokeDataTargetActionAuth(requestID, config, dataActionTargetAuthRequestBody)
		if err != nil {
			LogError(requestID, "Failed to authenticate Data Action Target request: "+err.Error())
			http.Error(incomingRespWriter, err.Error(), authResponseStatus)
			return false
		}
	}

	// Handle unauthorized or unexpected failed auth requests
	if authResponseStatus != http.StatusOK {
		if authResponseStatus == http.StatusUnauthorized || authResponseStatus == http.StatusForbidden {
			LogWarn(requestID, "Unauthorized request! "+unauthorizedMsg)
			// Unauthenticated requests that appear to be valid are 403 Forbidden
			http.Error(incomingRespWriter, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			incomingRespWriter.WriteHeader(authResponseStatus)
		} else {
			// Unexpected error
			LogError(requestID, "Failed to authenticate request: statusCode "+strconv.Itoa(authResponseStatus)+", body '"+authResponseBody+"'")
			http.Error(incomingRespWriter, authResponseBody, authResponseStatus)
			incomingRespWriter.WriteHeader(authResponseStatus)
		}

		// Do NOT forward
		return false
	}

	// Successful authentication!
	LogInfo(requestID, "Authenticated request!")
	return true
}

// InvokeSalesforceAuth Authenticate Salesforce request
func InvokeSalesforceAuth(requestID string, config *conf.Config, sfAuthRequestBody SalesforceAuthRequestBody) (int, string, error) {
	LogInfo(requestID, "Authenticating Salesforce request for org "+sfAuthRequestBody.OrgID+", domain "+
		sfAuthRequestBody.OrgDomainUrl+"...")

	operation := "Salesforce authentication"
	jsonBody, err := json.Marshal(sfAuthRequestBody)
	statusCode, body, err := InvokeHerokuIntegrationService(requestID, config, operation, config.HerokuInvocationSalesforceAuthPath,
		http.MethodPost, jsonBody)

	return statusCode, body, err
}

// InvokeDataTargetActionAuth Authenticate Data Action Target webhook request
func InvokeDataTargetActionAuth(requestID string, config *conf.Config, dataActionTargetAuthRequestBody DataActionTargetAuthRequestBody) (int, string, error) {
	LogInfo(requestID, "Authenticating Data Action Target '"+dataActionTargetAuthRequestBody.ApiName+"' request from org "+
		dataActionTargetAuthRequestBody.OrgID+" with payload length "+strconv.Itoa(len(dataActionTargetAuthRequestBody.Payload))+"...")

	operation := "Data Action Target authentication"
	jsonBody, err := json.Marshal(dataActionTargetAuthRequestBody)
	statusCode, body, err := InvokeHerokuIntegrationService(requestID, config, operation, config.HerokuIntegrationDataActionTargetAuthPath,
		http.MethodPost, jsonBody)

	return statusCode, body, err
}

// InvokeHerokuIntegrationService Invoke given Heroku Integration API with given request JSON body.
func InvokeHerokuIntegrationService(
	requestID string,
	config *conf.Config,
	operation string,
	apiPath string,
	httpMethod string,
	jsonBody []byte) (int, string, error) {
	defer TimeTrack(requestID, time.Now(), operation)

	statusCode := http.StatusInternalServerError
	body := ""

	integrationApiUrl := config.HerokuIntegrationUrl + apiPath
	req, err := http.NewRequest(httpMethod, integrationApiUrl, bytes.NewBuffer(jsonBody))
	if err != nil {
		LogError(requestID, fmt.Sprintf("Failed to assemble %s request: %v", operation, err))
		return statusCode, body, fmt.Errorf("unable to assemble %s request %s: %v", operation, requestID, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.HerokuInvocationToken)
	req.Header.Set("REQUEST_ID", requestID)

	// TODO: Remove when no longer needed
	LogDebug(requestID, fmt.Sprintf("!! REMOVEME !! Calling Heroku Integration API %s [%s] with body %s",
		integrationApiUrl, config.HerokuInvocationToken, jsonBody))

	// Invoke
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		LogError(requestID, fmt.Sprintf("Failed to invoke %s: %v", operation, err))
		return statusCode, body, fmt.Errorf("unable to invoke %s request %s: %v", operation, requestID, err)
	}

	defer resp.Body.Close()

	// Capture statusCode and body
	statusCode = resp.StatusCode
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		LogError(requestID, err.Error())
	} else {
		body = string(bodyBytes)
	}

	LogDebug(requestID, "Response for "+operation+" request ("+apiPath+"): statusCode "+strconv.Itoa(statusCode)+", body '"+body+"'")

	return statusCode, body, nil
}

// ForwardRequestReplyToIncomingRequest Forward request to target API; send response to incoming request
func ForwardRequestReplyToIncomingRequest(
	startTime time.Time,
	requestID string,
	forwardApiUrl string,
	incomingRespWriter http.ResponseWriter,
	incomingReq *http.Request,
	incomingReqBody []byte) {
	// Forward request to target API
	forwardResp := ForwardRequest(requestID, forwardApiUrl, incomingRespWriter, incomingReq, incomingReqBody)

	// Log time took to evaluate request and forward to API
	TimeTrack(requestID, startTime, "Heroku Integration Service Mesh")

	// Copy API's response to incoming response
	ReplyToIncomingRequest(requestID, forwardResp, incomingRespWriter)
}

// ForwardRequest Forward request to target API
func ForwardRequest(
	requestID string,
	forwardApiUrl string,
	incomingRespWriter http.ResponseWriter,
	incomingReq *http.Request,
	incomingReqBody []byte) *http.Response {

	// Forward request to target API

	LogInfo(requestID, "Forwarding request...")
	forwardReq, err := http.NewRequest(incomingReq.Method, forwardApiUrl, bytes.NewReader(incomingReqBody))
	if err != nil {
		LogError(requestID, "Failed to forward request: "+err.Error())
		http.Error(incomingRespWriter, err.Error(), http.StatusInternalServerError)
	}

	// Apply request headers to forward request
	for header, values := range incomingReq.Header {
		for _, value := range values {
			// Exclude x-request-context header
			if strings.EqualFold(header, HdrRequestContext) {
				continue
			}
			forwardReq.Header.Set(header, value)
		}
	}

	// Forward request
	client := &http.Client{}
	forwardResp, err := client.Do(forwardReq)
	if err != nil {
		LogError(requestID, "Failed to forward request: "+err.Error())
		http.Error(incomingRespWriter, "Failed to forward request "+requestID, http.StatusBadGateway)
	}

	return forwardResp
}

// ReplyToIncomingRequest Send API response to incoming response
func ReplyToIncomingRequest(requestID string, forwardResp *http.Response, incomingRespWriter http.ResponseWriter) {
	// Copy forwarded request's response headers to incoming response
	for header, values := range forwardResp.Header {
		for _, value := range values {
			incomingRespWriter.Header().Add(header, value)
		}
	}

	// Copy forward request's response to incoming response
	incomingRespWriter.WriteHeader(forwardResp.StatusCode)
	_, err := io.Copy(incomingRespWriter, forwardResp.Body)
	if err != nil {
		LogError(requestID, err.Error())
	}
	defer forwardResp.Body.Close()
}
