package mesh

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

const (
	HdrClientContext              = "x-client-context"
	HdrSignature                  = "x-signature"
	OrgIdQueryParam               = "orgId"
	ApiNameQueryParam             = "apiName"
	SalesforceExpectedHeaderCount = 2
)

type InvalidRequest struct {
	StatusCode int
	Err        error
}

func (e *InvalidRequest) HttpStatusCode() int {
	return e.StatusCode
}

func (e *InvalidRequest) Error() string {
	return fmt.Sprintf("%d %v", e.StatusCode, e.Err)
}

// NewInvalidRequest Return when request is invalid - 401 Unauthorized
func NewInvalidRequest(message string) *InvalidRequest {
	return &InvalidRequest{
		StatusCode: http.StatusUnauthorized,
		Err:        errors.New(message),
	}
}

// NewMalformedRequest Return when request is structured
// correctly - likely a valid Salesforce/Data Cloud request,
// but headers or header content is incorrect - 400 Bad Request
func NewMalformedRequest(message string) *InvalidRequest {
	return &InvalidRequest{
		StatusCode: http.StatusBadRequest,
		Err:        errors.New(message),
	}
}

type XRequestContext struct {
	ID           string `json:"id"`
	Auth         string `json:"auth"`
	LoginUrl     string `json:"loginUrl"`
	OrgDomainUrl string `json:"orgDomainUrl"`
	OrgID        string `json:"orgId"`
	Resource     string `json:"resource"`
	Type         string `json:"type"`
}

type RequestHeader struct {
	XRequestID          string          `json:"x-request-id"`
	XRequestContext     XRequestContext `json:"x-request-context"`
	XClientContext      string          `json:"x-client-context"`
	XSignature          string          `json:"x-signature"`
	IsSalesforceRequest bool            `json:"isDataActionTargetRequest"`
}

// ValidateRequest validates the request headers based on type - Salesforce or Data Action Target.
//
// Request Salesforce request headers:
//   - x-request-id
//   - x-request-context
//   - x-client-context
//
// Required Data Action Target request headers:
//   - x-request-id
//   - x-signature
func ValidateRequest(requestID string, headers http.Header) (*RequestHeader, error) {
	LogInfo(requestID, "Validating request...")

	// Log headers
	reqHeadersBytes, err := json.Marshal(headers)
	if err != nil {
		LogDebug(requestID, fmt.Sprintf("Could not marshal request headers: %v", err))
	} else {
		LogDebug(requestID, "Headers: "+string(reqHeadersBytes))
	}

	XRequestContextString := headers.Get(HdrRequestContext)
	xClientContext := headers.Get(HdrClientContext)
	xSignature := headers.Get(HdrSignature)

	// First check if Salesforce headers are present
	sfHeaderCount, sfHeaderErrors := doSalesforceHeadersExist(XRequestContextString, xClientContext)
	if sfHeaderCount != SalesforceExpectedHeaderCount {
		// There's an issue w/ Salesforce headers...
		if sfHeaderCount == 0 {
			// ZERO Salesforce headers were found.
			// Is this a Data Action Target request?
			if xSignature != "" {
				// Found Data Action Target request, no further validation here
				return &RequestHeader{
					XRequestID:          requestID,
					IsSalesforceRequest: false,
					XSignature:          xSignature,
				}, nil
			}

			// NOT a Salesforce, NOT a Data Action Target request
			LogDebug(requestID, "Errors: "+strings.Join(sfHeaderErrors, "; ")+"; Data Action Target header (x-signature) not found")
			LogError(requestID, "Invalid request!")
		} else {
			// Found some, but NOT all Salesforce headers
			LogDebug(requestID, "Errors: "+strings.Join(sfHeaderErrors, "; "))
			LogError(requestID, "Invalid request!")
			return nil, NewMalformedRequest("Invalid request")
		}

		return nil, NewInvalidRequest("Invalid request")
	}

	// Additional Salesforce header validation
	// 1. Decode the x-request-context
	contextData, err := base64.StdEncoding.DecodeString(XRequestContextString)
	if err != nil {
		LogError(requestID, "Invalid request! Unable to decode Salesforce "+HdrRequestContext+" header")
		return nil, NewMalformedRequest("Invalid " + HdrRequestContext + " header")
	}

	var XRequestContext XRequestContext
	if err := json.Unmarshal(contextData, &XRequestContext); err != nil {
		LogError(requestID, "Invalid request! Unable to unmarshal Salesforce "+HdrRequestContext+" header")
		return nil, NewMalformedRequest("Invalid " + HdrRequestContext + " header")
	}

	// 2. Ensure all values are present in request context
	if err := validateRequestContextValues(&XRequestContext); err != nil {
		LogError(requestID, "Invalid request! "+err.Error())
		return nil, err
	}

	// 3. Validate that x-request-id and x-request-context#id are the same
	if !strings.Contains(requestID, XRequestContext.ID) {
		LogError(requestID, "Invalid request! Missing or mismatch x-request-id and x-request-context#id")
		return nil, NewMalformedRequest("Invalid " + HdrRequestContext + " header")
	}

	LogInfo(requestID, "Valid request!")

	return &RequestHeader{
		XRequestID:          requestID,
		XRequestContext:     XRequestContext,
		XClientContext:      xClientContext,
		IsSalesforceRequest: true,
	}, nil
}

func doSalesforceHeadersExist(XRequestContext, xClientContext string) (int, []string) {
	sfHeaderCount := SalesforceExpectedHeaderCount
	var sfHeaderErrors []string

	if XRequestContext == "" {
		sfHeaderCount--
		sfHeaderErrors = append(sfHeaderErrors, "Invalid "+HdrRequestContext+" header")
	}

	if xClientContext == "" {
		sfHeaderCount--
		sfHeaderErrors = append(sfHeaderErrors, "Invalid "+HdrClientContext+" header")
	}

	return sfHeaderCount, sfHeaderErrors
}

func validateRequestContextValues(context *XRequestContext) error {
	v := reflect.ValueOf(*context)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).IsZero() {
			return NewMalformedRequest(fmt.Sprintf("Missing or invalid value in "+HdrRequestContext+" header: %s", v.Type().Field(i).Name))
		}
	}
	return nil
}
