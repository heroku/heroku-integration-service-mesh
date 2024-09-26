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

const SALESFORCE_EXPECTED_HEADER_COUNT = 2

const (
	HdrRequestsContext = "x-request-context"
	HdrClientContext   = "x-client-context"
	HdrSignature       = "x-signature"
	OrgIdQueryParm     = "orgId"
	ApiNameQueryParam  = "apiName"
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

// Return when request is invalid - 401 Unauthorized
func NewInvalidRequest(message string) *InvalidRequest {
	return &InvalidRequest{
		StatusCode: http.StatusUnauthorized,
		Err:        errors.New(message),
	}
}

// Return when request is structured correctly, but header content is incorrect - 400 Bad Request
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

/**
 * Validate the request headers based on type - Salesforce or Data Action Target.
 *
 * Request Salesforce request headers:
 *   - x-request-id
 *   - x-request-context
 *   - x-client-context
 *
 * Required Data Action Target request headers:
 *   - x-signature
 *
 * @param requestID
 * @param header
 * @return RequestHeader, error
 * @throws MeshValidationException
 * @throws InvalidRequest
 * @throws MalformedRequest
 */
func ValidateRequest(requestID string, headers http.Header) (*RequestHeader, error) {
	logInfo(requestID, "Validating request...")

	// Log headers
	reqHeadersBytes, err := json.Marshal(headers)
	if err != nil {
		logDebug(requestID, fmt.Sprintf("Could not marshal request headers: %v", err))
	} else {
		logDebug(requestID, "Headers: "+string(reqHeadersBytes))
	}

	XRequestContextString := headers.Get(HdrRequestsContext)
	xClientContext := headers.Get(HdrClientContext)
	xSignature := headers.Get(HdrSignature)

	// Fsirst check if the salesforce headers are present
	sfHeaderPresenceErrors := validatePresence(requestID, XRequestContextString, xClientContext)
	if sfHeaderPresenceErrors != nil && len(sfHeaderPresenceErrors) > 0 {
		// Found ISSUE w/ Salesforce headers found...
		if len(sfHeaderPresenceErrors) == SALESFORCE_EXPECTED_HEADER_COUNT {
			// ZERO Salesforce headers were found.  Is this a Data Action Targe request?
			if xSignature != "" {
				// Found Data Action Target request
				return &RequestHeader{
					XRequestID:          requestID,
					IsSalesforceRequest: false,
					XSignature:          xSignature,
				}, nil
			}

			// NOT a Salesforce OR a Data Action Target request
			logError(requestID, "Invalid request!  Invalid Salesforce header(s): "+strings.Join(sfHeaderPresenceErrors, ", "))
			logError(requestID, "Invalid request!  Invalid Data Action Target x-signature header")
		} else {
			// Found some, but NOT ALL Salesforce headers
			logError(requestID, "Invalid request!  Invalid Salesforce header(s): "+strings.Join(sfHeaderPresenceErrors, ", "))
			return nil, NewMalformedRequest("Invalid request")
		}

		return nil, NewInvalidRequest("Invalid request")
	}

	// Additional Salesforce header validation
	// 1. Decode the x-request-context
	contextData, err := base64.StdEncoding.DecodeString(XRequestContextString)
	if err != nil {
		logError(requestID, "Invalid request!  Unable to decode Salesforce x-request-context header")
		return nil, NewMalformedRequest("Invalid Salesforce x-request-context")
	}

	var XRequestContext XRequestContext
	if err := json.Unmarshal(contextData, &XRequestContext); err != nil {
		logError(requestID, "Invalid request!  Unable to unmarshal Salesforce x-request-context header")
		return nil, NewMalformedRequest("Invalid Salesforce x-request-context")
	}

	// 2. Ensure all values are present in request context
	if err := validateRequestContextValues(requestID, &XRequestContext); err != nil {
		logError(requestID, "Invalid request!  Unable to validate Salesforce x-request-context header: "+err.Error())
		return nil, err
	}

	// 3. Validate that request is coming from an org
	orgID := XRequestContext.OrgID
	// TODO: Adjust once both requestId and orgId are both 18-char
	truncatedOrgID := orgID[:len(orgID)-3]
	if !strings.Contains(requestID, truncatedOrgID) {
		// REVIEWME: If x-request-id header was missing, we generate one and set, so this will fail - warn?
		logError(requestID, "Invalid request!  Missing or mismatch orgId in Salesforce x-request-context header")
		return nil, NewMalformedRequest("Invalid Salesforce x-request-context")
	}

	logInfo(requestID, "Valid request!")

	return &RequestHeader{
		XRequestID:          requestID,
		XRequestContext:     XRequestContext,
		XClientContext:      xClientContext,
		IsSalesforceRequest: true,
	}, nil
}

func validatePresence(requestID, XRequestContext, xClientContext string) []string {
	var errors []string

	if XRequestContext == "" {
		errors = append(errors, "Invalid x-request-context header")
	}

	if xClientContext == "" {
		errors = append(errors, "Invalid x-client-context header")
	}

	return errors
}

func validateRequestContextValues(requestID string, context *XRequestContext) error {
	v := reflect.ValueOf(*context)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).IsZero() {
			NewMalformedRequest(fmt.Sprintf("Missing value for Salesforce x-request-context header request %s: %s", requestID, v.Type().Field(i).Name))
		}
	}
	return nil
}
