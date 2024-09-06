package mesh

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"main/conf"
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

	integrationToken := conf.GetConfig().InvocationToken
	fmt.Printf("Token: %s\n", integrationToken)

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

	var xRequestContext XRequestsContext
	if err := json.Unmarshal(contextData, &xRequestContext); err != nil {
		return nil, InvalidXRequestsContext
	}

	fmt.Printf("id: %s, auth: %s, loginUrl: %s, orgId: %s, orgDomainUrl: %s, resource: %s, type: %s\n", xRequestContext.ID, xRequestContext.Auth, xRequestContext.LoginUrl, xRequestContext.OrgDomainUrl, xRequestContext.OrgID, xRequestContext.Resource, xRequestContext.Type)

	//ensure all values are present in request context
	if err := validateRequestContextValues(&xRequestContext); err != nil {
		return nil, err
	}

	//validate that request is coming from an org
	orgID := xRequestContext.OrgID
	if !strings.Contains(xRequestID, orgID) {
		return nil, InvalidRequestId
	}

	return &RequestHeader{
		XRequestID:          xRequestID,
		XRequestContext:     XRequestsContext{},
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
