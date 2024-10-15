package test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
		Authentication: conf.Authentication{
			BypassRoutes: []string{"/byPassMe"},
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
	if !shouldBypass {
		t.Error("Should bypass")
	}

	shouldBypass = mesh.ShouldBypassValidationAuthentication(MockRequestID, config, "/byPassMe?moreStuffHere=true")
	if !shouldBypass {
		t.Error("Should bypass")
	}

	shouldBypass = mesh.ShouldBypassValidationAuthentication(MockRequestID, config, "/bypassme")
	if shouldBypass {
		t.Error("Should NOT have bypass")
	}

	yamlConfig = &conf.YamlConfig{}
	config = &conf.Config{
		ShouldBypassAllRoutes: false,
		YamlConfig:            yamlConfig,
	}
	shouldBypass = mesh.ShouldBypassValidationAuthentication(MockRequestID, config, "/bypassme")
	if shouldBypass {
		t.Error("Should NOT have bypass")
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
	config := &conf.Config{
		AppPort: urlParts[2],
		AppUrl:  urlParts[0] + ":" + urlParts[1],
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

	forwardApiUrl, err := mesh.GetForwardUrl(config.AppUrl, config.AppPort, incomingReq)
	mesh.ForwardRequestReplyToIncomingRequest(time.Now(), MockRequestID, forwardApiUrl, incomingRespWriter, incomingReq, jsonData)
	if err != nil {
		t.Fatal(err)
	}
}
