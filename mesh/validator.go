package mesh

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
)

const (
	HdrNameRequestID   = "x-request-id"
	HdrRequestsContext = "x-request-context"
	HdrClientContext   = "x-client-context"
	HdrSignature       = "x-signature"
	OrgIdQueryParm     = "orgId"
	ApiNameQueryParam  = "apiName"
)

var (
	MissingValuesError      = errors.New("x-request-id, x-requests-context, and x-client-context or x-signature are required")
	InvalidRequestId        = errors.New("invalid x-request-id")
	InvalidXRequestsContext = errors.New("invalid x-requests-context")
)

type XRequestsContext struct {
	ID           string `json:"id"`
	Auth         string `json:"auth"`
	LoginUrl     string `json:"loginUrl"`
	OrgDomainUrl string `json:"orgDomainUrl"`
	OrgID        string `json:"orgId"`
	Resource     string `json:"resource"`
	Type         string `json:"type"`
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

		slog.Error("Validation error: x-request-id, x-contexts-request, and x-client-context or x-signature are required")
		return nil, MissingValuesError
	}

	// decode the x-requests-context
	contextData, err := base64.StdEncoding.DecodeString(xRequestsContextString)
	if err != nil {
		slog.Error("Unable to decode x-requests-context")
		return nil, InvalidXRequestsContext
	}

	var xRequestContext XRequestsContext
	if err := json.Unmarshal(contextData, &xRequestContext); err != nil {
		slog.Error("Unable to unmarshal  x-requests-context")
		return nil, InvalidXRequestsContext
	}

	//ensure all values are present in request context
	if err := validateRequestContextValues(&xRequestContext); err != nil {
		slog.Error("Unable to validate x-requests-context: " + err.Error())
		return nil, err
	}

	//validate that request is coming from an org
	orgID := xRequestContext.OrgID
	truncatedOrgID := orgID[:len(orgID)-3]
	if !strings.Contains(xRequestID, truncatedOrgID) {
		slog.Error("Missing org id in x-request-id")
		return nil, InvalidRequestId
	}

	return &RequestHeader{
		XRequestID:          xRequestID,
		XRequestContext:     xRequestContext,
		XClientContext:      xClientContext,
		IsSalesforceRequest: true,
	}, nil
}

func validatePresence(xRequestID, xRequestContext, xClientContext string) bool {

	return xRequestID != "" && xRequestContext != "" && xClientContext != ""
}

func validateRequestContextValues(context *XRequestsContext) error {
	v := reflect.ValueOf(*context)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).IsZero() {
			return fmt.Errorf("missing value for x-requests-context: %s", v.Type().Field(i).Name)
		}
	}
	return nil
}
