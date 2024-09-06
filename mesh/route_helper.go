package mesh

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
)

const (
	HdrNameRequestID          = "x-request-id"
	HdrRequestsContext        = "x-requests-context"
	HdrClientContext          = "x-client-context"
	HdrSignature              = "x-signature"
	DataActionTargetQueryParm = "dat"
	OrgIdQueryParm            = "orgId"
)

var (
	MissingValuesError      = errors.New("x-request-id, x-requests-context, and x-client-context or x-signature are required")
	InvalidRequestId        = errors.New("invalid x-request-id")
	MissingKeyInContext     = errors.New("missing key in x-requests-context")
	InvalidXRequestsContext = errors.New("invalid x-requests-context")
)

type XRequestsContext struct {
	ID            string `json:"id"`
	Auth          string `json:"auth"`
	LoginUrl      string `json:"loginUrl"`
	OrgDomainUrl  string `json:"orgDomainUrl"`
	OrgID         string `json:"orgId"`
	Resource      string `json:"resource"`
	SchemaVersion string `json:"schemaVersion"`
	Source        string `json:"source"`
	Type          string `json:"type"`
}

type RequestHeader struct {
	XRequestID          string           `json:"x-request-id"`
	XRequestContext     XRequestsContext `json:"x-request-context"`
	XClientContext      string           `json:"x-client-context"`
	XSignature          string           `json:"x-signature"`
	IsSalesforceRequest bool             `json:"isDataCloudRequest"`
}

func ValidateRequest(header http.Header) (*RequestHeader, error) {
	xRequestID := header.Get(HdrNameRequestID)
	xRequestsContextString := header.Get(HdrRequestsContext)
	xClientContext := header.Get(HdrClientContext)
	xSignature := header.Get(HdrSignature)

	// first check if the salesforce headers are present, then check if the data-cloud header is present
	if !validatePresence(xRequestID, xRequestsContextString, xClientContext) {
		if xSignature != "" {
			return &RequestHeader{
				IsSalesforceRequest: false,
				XSignature:          xSignature,
			}, nil
		}

		return nil, MissingValuesError
	}

	// decode the x-requests-context
	contextData, err := base64.StdEncoding.DecodeString(xRequestsContextString)
	if err != nil {
		return nil, InvalidXRequestsContext
	}

	var xRequestsContext XRequestsContext
	if err := json.Unmarshal(contextData, &xRequestsContext); err != nil {
		return nil, InvalidXRequestsContext
	}

	// ensure all values are present in request context
	if err := validateRequestContextValues(&xRequestsContext); err != nil {
		return nil, err
	}

	// validate that request is coming from an org
	orgID := xRequestsContext.OrgID
	if !strings.Contains(xRequestID, orgID) {
		return nil, InvalidRequestId
	}

	return &RequestHeader{
		XRequestID:          xRequestID,
		XRequestContext:     xRequestsContext,
		XClientContext:      xClientContext,
		IsSalesforceRequest: true,
	}, nil
}

func validatePresence(xRequestID, xRequestContext, xClientContext string) bool {

	return xRequestID == "" || xRequestContext == "" || xClientContext == ""
}

func validateRequestContextValues(context *XRequestsContext) error {
	v := reflect.ValueOf(*context)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).IsZero() {
			return MissingKeyInContext
		}
	}
	return nil
}
