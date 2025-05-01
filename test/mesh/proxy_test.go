package mesh

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/heroku/heroku-integration-service-mesh/conf"
	"github.com/heroku/heroku-integration-service-mesh/mesh"
)

func Test_ShouldBypassValidationAuthentication(t *testing.T) {
	config := &conf.Config{
		ShouldBypassAllRoutes: true,
	}

	shouldBypass := mesh.ShouldBypassValidationAuthentication(MockRequestID, config, "")
	if !shouldBypass {
		t.Error("Should bypass ALL")
	}

	yamlConfig := &conf.YamlConfig{
		Mesh: conf.Mesh{
			Authentication: conf.Authentication{
				BypassRoutes: []string{"/byPassMe", "/favicon*"},
			},
			HealthCheck: conf.HealthCheck{
				Enable: "false",
			},
		},
	}
	config = &conf.Config{
		ShouldBypassAllRoutes: false,
		YamlConfig:            yamlConfig,
	}
	shouldBypass = mesh.ShouldBypassValidationAuthentication(MockRequestID, config, "/byPassMe")
	if !shouldBypass {
		t.Error("Should bypass")
	}

	shouldBypass = mesh.ShouldBypassValidationAuthentication(MockRequestID, config, "/byPassMe/moreStuffHere")
	if shouldBypass {
		t.Error("Should NOT bypass")
	}

	shouldBypass = mesh.ShouldBypassValidationAuthentication(MockRequestID, config, "/byPassMe?moreStuffHere=true")
	if !shouldBypass {
		t.Error("Should bypass")
	}

	shouldBypass = mesh.ShouldBypassValidationAuthentication(MockRequestID, config, "/favicon-32x32.png")
	if !shouldBypass {
		t.Error("Should bypass")
	}

	shouldBypass = mesh.ShouldBypassValidationAuthentication(MockRequestID, config, "/favicon/another.png")
	if !shouldBypass {
		t.Error("Should bypass")
	}

	shouldBypass = mesh.ShouldBypassValidationAuthentication(MockRequestID, config, "/favIcon-32x32.png")
	if shouldBypass {
		t.Error("Should NOT bypass")
	}

	shouldBypass = mesh.ShouldBypassValidationAuthentication(MockRequestID, config, "/bypassme")
	if shouldBypass {
		t.Error("Should NOT bypass")
	}

	yamlConfig = &conf.YamlConfig{}
	config = &conf.Config{
		ShouldBypassAllRoutes: false,
		YamlConfig:            yamlConfig,
	}
	shouldBypass = mesh.ShouldBypassValidationAuthentication(MockRequestID, config, "/bypassme")
	if shouldBypass {
		t.Error("Should NOT bypass")
	}

	shouldBypass = mesh.ShouldBypassValidationAuthentication(MockRequestID, config, conf.HealthCheckRoute)
	if shouldBypass {
		t.Error("Should NOT bypass")
	}

	yamlConfig = &conf.YamlConfig{
		Mesh: conf.Mesh{
			HealthCheck: conf.HealthCheck{
				Enable: "true",
				Route:  "/healthcheck",
			},
		},
	}
	config = &conf.Config{
		ShouldBypassAllRoutes: false,
		YamlConfig:            yamlConfig,
	}
	shouldBypass = mesh.ShouldBypassValidationAuthentication(MockRequestID, config, yamlConfig.Mesh.HealthCheck.Route)
	if !shouldBypass {
		t.Error("Should bypass")
	}

	yamlConfig = &conf.YamlConfig{
		Mesh: conf.Mesh{
			HealthCheck: conf.HealthCheck{
				Enable: "false",
				Route:  "/healthcheck",
			},
		},
	}
	config = &conf.Config{
		ShouldBypassAllRoutes: false,
		YamlConfig:            yamlConfig,
	}
	shouldBypass = mesh.ShouldBypassValidationAuthentication(MockRequestID, config, yamlConfig.Mesh.HealthCheck.Route)
	if shouldBypass {
		t.Error("Should NOT bypass")
	}
}

func Test_SalesforceAuth(t *testing.T) {
	herokuInvocationToken := "HerokuInvocationToken"
	auth := "auth"

	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.URL.Path != conf.HerokuIntegrationSalesforceAuthPath {
			t.Errorf("Expected to request path "+conf.HerokuIntegrationSalesforceAuthPath+", got '%s'", request.URL.Path)
		}

		if request.Method != "POST" {
			t.Errorf("Expected to request POST, got '%s'", request.Method)
		}

		if request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type 'application/json' header, got '%s'", request.Header.Get("Content-Type"))
		}

		if request.Header.Get("Authorization") != "Bearer "+herokuInvocationToken {
			t.Errorf("Expected Authorization 'Bearer "+herokuInvocationToken+"' header, got '%s'", request.Header.Get("Authorization"))
		}

		if request.Header.Get("REQUEST_ID") != MockRequestID {
			t.Errorf("Expected REQUEST_ID '"+MockRequestID+"' header, got '%s'", request.Header.Get("REQUEST_ID"))
		}

		// Get the request body from the incoming request
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Error(err.Error())
		}

		// Unmarshal the JSON data
		var salesforceAuthRequestBody mesh.SalesforceAuthRequestBody
		err = json.Unmarshal(body, &salesforceAuthRequestBody)
		if err != nil {
			t.Error(err.Error())
		}
		if salesforceAuthRequestBody.CoreJWTToken != auth {
			t.Errorf("Expected CoreJWTToken to be '"+MockRequestID+"', got '%s'", salesforceAuthRequestBody.CoreJWTToken)
		}
		// TODO: Validate the rest of SalesforceAuthRequestBody payload

		responseWriter.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &conf.Config{
		HerokuIntegrationUrl:               server.URL,
		HerokuInvocationToken:              herokuInvocationToken,
		HerokuInvocationSalesforceAuthPath: conf.HerokuIntegrationSalesforceAuthPath,
	}
	requestHeader := &mesh.RequestHeader{
		XRequestID: MockRequestID,
		XRequestContext: mesh.XRequestContext{
			ID:           MockRequestID,
			Auth:         auth,
			LoginUrl:     "http://login.salesforce.com",
			OrgDomainUrl: "http://org.salesforce.com",
			OrgID:        MockOrgID18,
			Resource:     "resource",
			Type:         "type",
			AppUUID:      MockUUID,
		},
		XClientContext:      MockRequestID,
		IsSalesforceRequest: true,
	}
	incomingReq, err := http.NewRequest("POST", "/my-api", nil)
	if err != nil {
		t.Fatal(err)
	}
	incomingRespWriter := httptest.NewRecorder()
	incomingReqBody := make([]byte, 0)

	isAuth := mesh.AuthenticateRequest(MockRequestID, config, requestHeader, incomingRespWriter, incomingReq, incomingReqBody)
	if !isAuth {
		t.Errorf("Expected authenticated request")
	}
}

func Test_DataActionTargetAuth(t *testing.T) {
	herokuInvocationToken := "HerokuInvocationToken"

	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.URL.Path != conf.HerokuIntegrationDataActionTargetAuthPath {
			t.Errorf("Expected to request path "+conf.HerokuIntegrationDataActionTargetAuthPath+", got '%s'", request.URL.Path)
		}

		if request.Method != "POST" {
			t.Errorf("Expected to request POST, got '%s'", request.Method)
		}

		if request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type 'application/json' header, got '%s'", request.Header.Get("Content-Type"))
		}

		if request.Header.Get("Authorization") != "Bearer "+herokuInvocationToken {
			t.Errorf("Expected Authorization 'Bearer "+herokuInvocationToken+"' header, got '%s'", request.Header.Get("Authorization"))
		}

		if request.Header.Get("REQUEST_ID") != MockRequestID {
			t.Errorf("Expected REQUEST_ID '"+MockRequestID+"' header, got '%s'", request.Header.Get("REQUEST_ID"))
		}

		// Get the request body from the incoming request
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Error(err.Error())
		}

		// Unmarshal the JSON data
		var dataActionTargetAuthRequestBody mesh.DataActionTargetAuthRequestBody
		err = json.Unmarshal(body, &dataActionTargetAuthRequestBody)
		if err != nil {
			t.Error(err.Error())
		}
		if dataActionTargetAuthRequestBody.Signature != MockRequestID {
			t.Errorf("Expected Signature to be '"+MockRequestID+"', got '%s'", dataActionTargetAuthRequestBody.Signature)
		}
		// TODO: Validate the rest of DataActionTargetAuthRequestBody payload

		responseWriter.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &conf.Config{
		HerokuIntegrationUrl:                      server.URL,
		HerokuInvocationToken:                     herokuInvocationToken,
		HerokuIntegrationDataActionTargetAuthPath: conf.HerokuIntegrationDataActionTargetAuthPath,
	}
	requestHeader := &mesh.RequestHeader{
		XRequestID:          MockRequestID,
		XSignature:          MockRequestID,
		IsSalesforceRequest: false,
	}
	incomingReq, err := http.NewRequest("POST", "/my-api", nil)
	if err != nil {
		t.Fatal(err)
	}
	incomingRespWriter := httptest.NewRecorder()
	incomingReqBody := make([]byte, 0)

	isAuth := mesh.AuthenticateRequest(MockRequestID, config, requestHeader, incomingRespWriter, incomingReq, incomingReqBody)
	if !isAuth {
		t.Errorf("Expected authenticated request")
	}
}

func Test_ForwardRequestSendResponse(t *testing.T) {
	apiPath := "/my-api"

	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.URL.Path != apiPath {
			t.Errorf("Expected to request path "+apiPath+", got '%s'", request.URL.Path)
		}

		if request.Method != "POST" {
			t.Errorf("Expected to request POST, got '%s'", request.Method)
		}

		if request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type 'application/json' header, got '%s'", request.Header.Get("Content-Type"))
		}

		if request.Header.Get(mesh.HdrNameRequestID) != MockRequestID {
			t.Errorf("Expected "+mesh.HdrNameRequestID+" '"+MockRequestID+"' header, got '%s'", request.Header.Get(mesh.HdrNameRequestID))
		}

		if request.Header.Get(mesh.HdrRequestContext) != "" {
			t.Errorf("Expected "+mesh.HdrRequestContext+" '' header, got '%s'", request.Header.Get(mesh.HdrRequestContext))
		}

		if request.Header.Get(mesh.HdrClientContext) != MockRequestID {
			t.Errorf("Expected "+mesh.HdrClientContext+" '"+MockRequestID+"' header, got '%s'", request.Header.Get(mesh.HdrClientContext))
		}

		// Get the request body from the incoming request
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Error(err.Error())
		}

		// Unmarshal the JSON data
		var data map[string]interface{}
		err = json.Unmarshal(body, &data)
		if err != nil {
			t.Error(err.Error())
		}
		if data["name"] != "John" {
			t.Errorf("Expected name to be 'John', got '%s'", data["name"])
		}

		responseWriter.WriteHeader(http.StatusOK)
		responseWriter.Write([]byte(`{"hello":"there"}`))
	}))
	defer server.Close()

	urlParts := strings.Split(server.URL, ":")
	yamlConfig := &conf.YamlConfig{
		App: conf.App{
			Host: urlParts[0] + ":" + urlParts[1],
			Port: urlParts[2],
		},
	}
	config := &conf.Config{
		YamlConfig: yamlConfig,
	}
	jsonData := []byte(`{"name": "John", "age": 30}`)
	incomingReq, err := http.NewRequest("POST", apiPath, bytes.NewReader(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	incomingReq.Header.Set(mesh.HdrNameRequestID, MockRequestID)
	incomingReq.Header.Set(mesh.HdrRequestContext, mesh.HdrRequestContext)
	incomingReq.Header.Set(mesh.HdrClientContext, MockRequestID)
	incomingReq.Header.Set("Content-Type", "application/json")
	incomingRespWriter := httptest.NewRecorder()

	forwardApiUrl, err := mesh.GetForwardUrl(config.YamlConfig.App.Host, config.YamlConfig.App.Port, incomingReq)
	mesh.ForwardRequestReplyToIncomingRequest(time.Now(), MockRequestID, forwardApiUrl, incomingRespWriter, incomingReq, jsonData)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_ForwardRequestToUnavailableService(t *testing.T) {
	apiPath := "/my-api"

	jsonData := []byte(`{"name": "John", "age": 30}`)
	incomingReq, err := http.NewRequest("POST", apiPath, bytes.NewReader(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	incomingReq.Header.Set(mesh.HdrNameRequestID, MockRequestID)
	incomingReq.Header.Set(mesh.HdrRequestContext, mesh.HdrRequestContext)
	incomingReq.Header.Set(mesh.HdrClientContext, MockRequestID)
	incomingReq.Header.Set("Content-Type", "application/json")
	incomingRespWriter := httptest.NewRecorder()

	invalidServerUrl := "invalid"

	mesh.ForwardRequestReplyToIncomingRequest(time.Now(), MockRequestID, invalidServerUrl, incomingRespWriter, incomingReq, jsonData)
	if err != nil {
		t.Fatal(err)
	}
	responseStatus := incomingRespWriter.Result().StatusCode
	if responseStatus != http.StatusBadGateway {
		t.Fatal(fmt.Errorf("unexpected response for an invalid forward URL. Expected: %d, actual %d", http.StatusBadGateway, responseStatus))
	}
}

func Test_GetIntegrationURLForAddonUUID(t *testing.T) {
	// Set up test environment variables
	originalEnv := os.Environ()
	defer func() {
		// Restore original environment
		os.Clearenv()
		for _, env := range originalEnv {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
	}()

	expectedURL := "https://applink.heroku.com/addons/" + MockUUID + "/connections/salesforce"

	testCases := []struct {
		name          string
		envVars       map[string]string
		expectedURL   string
		expectedError bool
	}{
		{
			name:          "No matching environment variable",
			envVars:       map[string]string{},
			expectedURL:   "",
			expectedError: true,
		},
		{
			name: "Matching environment variable exists",
			envVars: map[string]string{
				"HEROKU_INTEGRATION_URL": expectedURL,
			},
			expectedURL:   expectedURL,
			expectedError: false,
		},
		{
			name: "Multiple environment variables, one matching",
			envVars: map[string]string{
				"OTHER_VAR":              "some value",
				"HEROKU_INTEGRATION_URL": expectedURL,
				"ANOTHER_VAR":            "another value",
			},
			expectedURL:   expectedURL,
			expectedError: false,
		},
		{
			name: "Environment variable with different name but correct URL format",
			envVars: map[string]string{
				"SOME_OTHER_VAR": expectedURL,
			},
			expectedURL:   expectedURL,
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear environment and set test variables
			os.Clearenv()
			for key, value := range tc.envVars {
				os.Setenv(key, value)
			}

			// Run test
			url, err := mesh.GetIntegrationURLForAddonUUID(MockUUID)

			// Check error
			if tc.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Check URL
			if url != tc.expectedURL {
				t.Errorf("Expected URL %s, got %s", tc.expectedURL, url)
			}
		})
	}
}

func Test_GetAddonUUIDFromRequestContext(t *testing.T) {
	// Create base context data
	baseContext := mesh.XRequestContext{
		ID:           MockRequestID,
		Auth:         "auth",
		LoginUrl:     "http://login.salesforce.com",
		OrgDomainUrl: "http://org.salesforce.com",
		OrgID:        MockOrgID18,
		Resource:     "resource",
		Type:         "type",
	}

	testCases := []struct {
		name          string
		headerValue   string
		contextData   *mesh.XRequestContext
		expectedUUID  string
		expectedError bool
	}{
		{
			name:          "Missing header",
			headerValue:   "",
			contextData:   nil,
			expectedUUID:  "",
			expectedError: true,
		},
		{
			name:          "Invalid base64 encoding",
			headerValue:   "invalid-base64",
			contextData:   nil,
			expectedUUID:  "",
			expectedError: true,
		},
		{
			name:          "Invalid JSON",
			headerValue:   base64.StdEncoding.EncodeToString([]byte("invalid-json")),
			contextData:   nil,
			expectedUUID:  "",
			expectedError: true,
		},
		{
			name:          "Missing AddonUUID",
			headerValue:   "",
			contextData:   &baseContext, // AddonUUID intentionally omitted
			expectedUUID:  "",
			expectedError: true,
		},
		{
			name:        "Valid request",
			headerValue: "",
			contextData: func() *mesh.XRequestContext {
				context := baseContext
				context.AddonUUID = MockUUID
				return &context
			}(),
			expectedUUID:  MockUUID,
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create request
			req, err := http.NewRequest("GET", "/test", nil)
			if err != nil {
				t.Fatal(err)
			}

			// Set header value
			if tc.contextData != nil {
				jsonData, err := json.Marshal(tc.contextData)
				if err != nil {
					t.Fatal(err)
				}
				tc.headerValue = base64.StdEncoding.EncodeToString(jsonData)
			}
			req.Header.Set(mesh.HdrRequestContext, tc.headerValue)

			// Run test
			uuid, err := mesh.GetAddonUUIDFromRequestContext(MockRequestID, req)

			// Check error
			if tc.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Check UUID
			if uuid != tc.expectedUUID {
				t.Errorf("Expected UUID %s, got %s", tc.expectedUUID, uuid)
			}
		})
	}
}
