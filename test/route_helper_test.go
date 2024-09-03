package test

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"net/http"
	"testing"

	mesh "main/mesh"
)

var MockID = uuid.New().String()
var MockOrgID = uuid.New().String()
var MockValidXRequestsContext = &mesh.XRequestsContext{
	ID:            MockID,
	Author:        "auth",
	LoginUrl:      "http://login.salesforce.com",
	OrgDomainUrl:  "http://org.salesforce.com",
	OrgID:         MockOrgID,
	Resource:      "resource",
	SchemaVersion: "1.0",
	Source:        "src",
	Type:          "type",
}

func MockXRequestsContextString() string {
	validXRequestsContextJSON, err := json.Marshal(MockValidXRequestsContext)
	if err != nil {
		return ""
	}
	return string(validXRequestsContextJSON)
}

func TestValidateRequestSuccess(t *testing.T) {

	header := http.Header{}
	header.Set(mesh.HdrNameRequestID, MockValidXRequestsContext.OrgID)
	header.Set(mesh.HdrRequestsContext, MockXRequestsContextString())
	header.Set(mesh.HdrClientContext, MockValidXRequestsContext.ID)

	_, err := mesh.ValidateRequest(header)
	if err != nil {
		t.Error(err)
	}
}

func TestValidateRequestFailureMissingHeaderKey(t *testing.T) {

	validXRequestsContextJSON, err := json.Marshal(MockValidXRequestsContext)
	if err != nil {
		t.Errorf("Error marshalling validXRequestsContext")
	}

	header := http.Header{}
	header.Set("x-request-id", MockValidXRequestsContext.OrgID)
	header.Set("x-requests-context", string(validXRequestsContextJSON))

	_, err = mesh.ValidateRequest(header)
	if !errors.Is(err, mesh.MissingValuesError) {
		t.Errorf("Expected '%v' got %v", mesh.MissingValuesError, err)
	}
}

func TestValidateRequestFailureMissingContextKey(t *testing.T) {
	invalidXRequestsContext := &mesh.XRequestsContext{
		ID:            uuid.New().String(),
		Author:        "auth",
		LoginUrl:      "http://login.salesforce.com",
		OrgDomainUrl:  "http://org.salesforce.com",
		OrgID:         uuid.New().String(),
		Resource:      "resource",
		SchemaVersion: "1.0",
		Source:        "src",
	}

	invalidXRequestsContextJSON, err := json.Marshal(invalidXRequestsContext)
	if err != nil {
		t.Errorf("Error marshalling validXRequestsContext")
	}

	header := http.Header{}
	header.Set("x-request-id", invalidXRequestsContext.OrgID)
	header.Set("x-requests-context", string(invalidXRequestsContextJSON))
	header.Set("x-client-context", invalidXRequestsContext.ID)

	_, err = mesh.ValidateRequest(header)
	if !errors.Is(err, mesh.MissingKeyInContext) {
		t.Errorf("Expected '%v' got %v", mesh.MissingKeyInContext, err)
	}

}
