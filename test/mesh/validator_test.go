package mesh

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/heroku/heroku-integration-service-mesh/mesh"
)

func Test_ValidateRequest_Success(t *testing.T) {
	headers := http.Header{}
	headers.Set(mesh.HdrNameRequestID, MockRequestID)
	headers.Set(mesh.HdrRequestContext, ConvertContextToString(MockValidXRequestContext))
	headers.Set(mesh.HdrClientContext, MockValidXRequestContext.ID)

	validRequestHeader, err := mesh.ValidateRequest(MockRequestID, headers)

	if validRequestHeader.XRequestID != MockRequestID {
		t.Error("Should have requestID")
	}

	if err != nil {
		t.Error(err.Error())
	}

	if !validRequestHeader.IsSalesforceRequest {
		t.Error("IsSalesforceRequest should be true")
	}
}

func Test_ValidateRequest_NoExpectedHeaders(t *testing.T) {
	headers := http.Header{}

	_, err := mesh.ValidateRequest(MockRequestID, headers)

	if err == nil {
		t.Error("Expected error")
	}

	_, ok := err.(*mesh.InvalidRequest)
	if !ok {
		t.Errorf("Expected mesh.InvalidRequest, got %v", err)
	}

	if err.(*mesh.InvalidRequest).HttpStatusCode() != http.StatusUnauthorized {
		t.Errorf("Expected %d, got %d", http.StatusUnauthorized, err.(*mesh.InvalidRequest).HttpStatusCode())
	}
}

func Test_InvalidRequestID(t *testing.T) {
	headers := http.Header{}
	headers.Set(mesh.HdrRequestContext, ConvertContextToString(MockValidXRequestContext))
	headers.Set(mesh.HdrClientContext, MockValidXRequestContext.ID)

	_, err := mesh.ValidateRequest("INVALID-REQUEST-ID", headers)

	if err == nil {
		t.Error("Expected error")
	}

	_, ok := err.(*mesh.InvalidRequest)
	if !ok {
		t.Errorf("Expected mesh.InvalidRequest, got %v", err)
	}

	if err.(*mesh.InvalidRequest).HttpStatusCode() != http.StatusBadRequest {
		t.Errorf("Expected %d, got %d", http.StatusBadRequest, err.(*mesh.InvalidRequest).HttpStatusCode())
	}
}

func Test_ValidateRequest_MissingRequestContextHeader(t *testing.T) {
	headers := http.Header{}
	headers.Set(mesh.HdrNameRequestID, MockValidXRequestContext.OrgID)
	headers.Set(mesh.HdrClientContext, MockValidXRequestContext.ID)

	_, err := mesh.ValidateRequest(MockRequestID, headers)

	if err == nil {
		t.Error("Expected error")
	}

	_, ok := err.(*mesh.InvalidRequest)
	if !ok {
		t.Errorf("Expected mesh.InvalidRequest, got %v", err)
	}

	if err.(*mesh.InvalidRequest).HttpStatusCode() != http.StatusBadRequest {
		t.Errorf("Expected %d, got %d", http.StatusBadRequest, err.(*mesh.InvalidRequest).HttpStatusCode())
	}
}

func Test_ValidateRequest_MissingClientContextHeader(t *testing.T) {
	headers := http.Header{}
	headers.Set(mesh.HdrNameRequestID, MockValidXRequestContext.OrgID)
	headers.Set(mesh.HdrRequestContext, ConvertContextToString(MockValidXRequestContext))

	_, err := mesh.ValidateRequest(MockRequestID, headers)

	if err == nil {
		t.Error("Expected error")
	}

	_, ok := err.(*mesh.InvalidRequest)
	if !ok {
		t.Errorf("Expected mesh.InvalidRequest, got %v", err)
	}

	if err.(*mesh.InvalidRequest).HttpStatusCode() != http.StatusBadRequest {
		t.Errorf("Expected %d, got %d", http.StatusBadRequest, err.(*mesh.InvalidRequest).HttpStatusCode())
	}
}

func Test_ValidateRequest_IncompleteRequestContext(t *testing.T) {
	invalidXRequestContext := &mesh.XRequestContext{
		ID:           uuid.New().String(),
		Auth:         "auth",
		LoginUrl:     "http://login.salesforce.com",
		OrgDomainUrl: "http://org.salesforce.com",
		OrgID:        uuid.New().String(),
	}

	headers := http.Header{}
	headers.Set(mesh.HdrNameRequestID, MockRequestID)
	headers.Set(mesh.HdrRequestContext, ConvertContextToString(invalidXRequestContext))
	headers.Set(mesh.HdrClientContext, invalidXRequestContext.ID)

	_, err := mesh.ValidateRequest(MockRequestID, headers)
	if err == nil {
		t.Errorf("Expected 'Missing or mismatch x-request-id and x-request-context#id', got %v", err)
	}
}

func Test_IsDataActionTargetRequest(t *testing.T) {
	headers := http.Header{}
	headers.Set(mesh.HdrSignature, uuid.New().String())

	validRequestHeader, err := mesh.ValidateRequest(MockRequestID, headers)

	if validRequestHeader.XRequestID != MockRequestID {
		t.Error("Should have requestID")
	}

	if err != nil {
		t.Error(err)
	}

	if validRequestHeader.IsSalesforceRequest {
		t.Error("IsSalesforceRequest should be false")
	}
}
