package test

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"testing"

	mesh "main/mesh"
)

var MockID = uuid.New().String()
var MockOrgID = uuid.New().String()
var MockValidXRequestsContext = &mesh.XRequestsContext{
	ID:           MockID,
	Auth:         "auth",
	LoginUrl:     "http://login.salesforce.com",
	OrgDomainUrl: "http://org.salesforce.com",
	OrgID:        MockOrgID,
	Resource:     "resource",
	Type:         "type",
}

func convertContextToString(context *mesh.XRequestsContext) string {

	requestContextJson, err := json.Marshal(context)
	if err != nil {
		fmt.Errorf(err.Error())
	}

	return base64.StdEncoding.EncodeToString(requestContextJson)
}

func TestValidateRequestSuccess(t *testing.T) {

	header := http.Header{}
	header.Set(mesh.HdrNameRequestID, MockValidXRequestsContext.OrgID)
	header.Set(mesh.HdrRequestsContext, convertContextToString(MockValidXRequestsContext))
	header.Set(mesh.HdrClientContext, MockValidXRequestsContext.ID)

	_, err := mesh.ValidateRequest(header)
	if err != nil {
		t.Error(err)
	}
}

func TestInvalidRequestID(t *testing.T) {

	header := http.Header{}
	header.Set(mesh.HdrNameRequestID, MockValidXRequestsContext.ID)
	header.Set(mesh.HdrRequestsContext, convertContextToString(MockValidXRequestsContext))
	header.Set(mesh.HdrClientContext, MockValidXRequestsContext.ID)

	_, err := mesh.ValidateRequest(header)
	if !errors.Is(err, mesh.InvalidRequestId) {
		t.Errorf("Expected '%v' got %v", mesh.InvalidRequestId, err)
	}
}

func TestValidateRequestFailureMissingHeaderKey(t *testing.T) {
	header := http.Header{}
	header.Set(mesh.HdrNameRequestID, MockValidXRequestsContext.OrgID)
	header.Set(mesh.HdrRequestsContext, convertContextToString(MockValidXRequestsContext))

	_, err := mesh.ValidateRequest(header)
	if !errors.Is(err, mesh.MissingValuesError) {
		t.Errorf("Expected '%v' got %v", mesh.MissingValuesError, err)
	}
}

func TestValidateRequestFailureMissingContextKey(t *testing.T) {
	invalidXRequestsContext := &mesh.XRequestsContext{
		ID:           uuid.New().String(),
		Auth:         "auth",
		LoginUrl:     "http://login.salesforce.com",
		OrgDomainUrl: "http://org.salesforce.com",
		OrgID:        uuid.New().String(),
	}

	header := http.Header{}
	header.Set(mesh.HdrNameRequestID, invalidXRequestsContext.OrgID)
	header.Set(mesh.HdrRequestsContext, convertContextToString(invalidXRequestsContext))
	header.Set(mesh.HdrClientContext, invalidXRequestsContext.ID)

	_, err := mesh.ValidateRequest(header)
	if err == nil {
		t.Errorf("Expected 'missing value for x-requests-context: Resource' got %v", err)
	}

}
