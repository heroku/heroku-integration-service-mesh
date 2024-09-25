package test

import (
	"encoding/base64"
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
	"testing"

	mesh "main/mesh"
)


var MockOrgID18 = "001Ws00003GGHVDIA5"
var MockUUID = uuid.New().String()
var MockRequestID =  MockOrgID18[:len(MockOrgID18)-3] + "-" + MockUUID
var MockValidXRequestContext = &mesh.XRequestContext{
	ID:           MockRequestID,
	Auth:         "auth",
	LoginUrl:     "http://login.salesforce.com",
	OrgDomainUrl: "http://org.salesforce.com",
	OrgID:        MockOrgID18,
	Resource:     "resource",
	Type:         "type",
}

func convertContextToString(context *mesh.XRequestContext) string {
	requestContextJson, err := json.Marshal(context)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(requestContextJson)
}

func TestValidateRequestSuccess(t *testing.T) {
	header := http.Header{}
	header.Set(mesh.HdrNameRequestID, MockRequestID)
	header.Set(mesh.HdrRequestsContext, convertContextToString(MockValidXRequestContext))
	header.Set(mesh.HdrClientContext, MockValidXRequestContext.ID)

	requestID, validateRequestHeader, err := mesh.ValidateRequest(MockRequestID, header)

	if requestID != MockRequestID {
		t.Error("Should have requestID")
	}

	if err != nil {
		t.Error(err.Error())
	}

	if !validateRequestHeader.IsSalesforceRequest {
		t.Error("IsSalesforceRequest should be true")
	}
}

func TestInvalidRequestID(t *testing.T) {
	header := http.Header{}
	header.Set(mesh.HdrNameRequestID, "INVALID-REQUEST-ID")
	header.Set(mesh.HdrRequestsContext, convertContextToString(MockValidXRequestContext))
	header.Set(mesh.HdrClientContext, MockValidXRequestContext.ID)

	_, _, err := mesh.ValidateRequest(MockRequestID, header)

	if err == nil {
		t.Error("Expected error")
	}

	if err.(*mesh.InvalidRequest).HttpStatusCode() != http.StatusBadRequest {
		t.Errorf("Expected %d, got %d", http.StatusBadRequest, err.(*mesh.InvalidRequest).HttpStatusCode())
	}
}

func TestValidateRequestFailureMissingHeaderKey(t *testing.T) {
	header := http.Header{}
	header.Set(mesh.HdrNameRequestID, MockValidXRequestContext.OrgID)
	header.Set(mesh.HdrRequestsContext, convertContextToString(MockValidXRequestContext))

	_, _, err := mesh.ValidateRequest(MockRequestID, header)

	if err == nil {
		t.Error("Expected error")
	}

	if err.(*mesh.InvalidRequest).HttpStatusCode() != http.StatusBadRequest {
		t.Errorf("Expected %d, got %d", http.StatusBadRequest, err.(*mesh.InvalidRequest).HttpStatusCode())
	}
}

func TestValidateRequestFailureMissingContextKey(t *testing.T) {
	invalidXRequestContext := &mesh.XRequestContext{
		ID:           uuid.New().String(),
		Auth:         "auth",
		LoginUrl:     "http://login.salesforce.com",
		OrgDomainUrl: "http://org.salesforce.com",
		OrgID:        uuid.New().String(),
	}

	header := http.Header{}
	header.Set(mesh.HdrNameRequestID, MockValidXRequestContext.OrgID)
	header.Set(mesh.HdrRequestsContext, convertContextToString(invalidXRequestContext))
	header.Set(mesh.HdrClientContext, invalidXRequestContext.ID)

	_, _, err := mesh.ValidateRequest(MockRequestID, header)
	if err == nil {
		t.Errorf("Expected 'missing value for x-requests-context: Resource' got %v", err)
	}

}

func TestIsDataActionTargetRequest(t *testing.T) {
	header := http.Header{}
	header.Set(mesh.HdrSignature, uuid.New().String())

	requestID, validateRequestHeader, err := mesh.ValidateRequest(MockRequestID, header)

	if requestID != MockRequestID {
		t.Error("Should have requestID")
	}

	if err != nil {
		t.Error(err)
	}

	if validateRequestHeader.IsSalesforceRequest {
		t.Error("IsSalesforceRequest should be false")
	}
}
