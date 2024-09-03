package mesh

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

const (
	HdrNameRequestID   = "x-request-id"
	HdrRequestsContext = "x-requests-context"
	HdrClientContext   = "x-client-context"
)

var (
	MissingValuesError  = errors.New("x-request-id, x-requests-context, and x-client-context are required")
	InvalidRequestId    = errors.New("invalid x-request-id")
	MissingKeyInContext = errors.New("missing key in x-requests-context")
)

type XRequestsContext struct {
	ID            string `json:"id"`
	Author        string `json:"authorization"`
	LoginUrl      string `json:"loginUrl"`
	OrgDomainUrl  string `json:"orgDomainUrl"`
	OrgID         string `json:"orgId"`
	Resource      string `json:"resource"`
	SchemaVersion string `json:"schemaVersion"`
	Source        string `json:"source"`
	Type          string `json:"type"`
}

type RequestHeader struct {
	XRequestID      string           `json:"x-Request-id"`
	XRequestContext XRequestsContext `json:"x-request-context"`
	XClientContext  string           `json:"x-client-context"`
}

func ValidateRequest(header http.Header) (*RequestHeader, error) {
	xRequestID := header.Get(HdrNameRequestID)
	xRequestsContextString := header.Get(HdrRequestsContext)
	xClientContext := header.Get(HdrClientContext)

	// check if all values are available and present
	if xRequestID == "" || xRequestsContextString == "" || xClientContext == "" {
		return nil, MissingValuesError
	}

	// decode the x-requests-context
	var xRequestsContext XRequestsContext
	if err := json.Unmarshal([]byte(xRequestsContextString), &xRequestsContext); err != nil {
		return nil, fmt.Errorf("invalid x-requests-context: %w", err)
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
		XRequestID:      xRequestID,
		XRequestContext: xRequestsContext,
		XClientContext:  xClientContext,
	}, nil
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
