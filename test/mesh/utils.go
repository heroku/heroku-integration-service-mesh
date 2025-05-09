package mesh

import (
	"encoding/base64"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/heroku/heroku-applink-service-mesh/mesh"
)

var MockOrgID18 = "00Dxx0000000000EAA"
var MockUUID = uuid.New().String()
var MockRequestID = MockOrgID18 + "-" + MockUUID
var MockValidXRequestContext = &mesh.XRequestContext{
	ID:           MockRequestID,
	Auth:         "auth",
	LoginUrl:     "http://login.salesforce.com",
	OrgDomainUrl: "http://org.salesforce.com",
	OrgID:        MockOrgID18,
	Resource:     "resource",
	Type:         "type",
	AppUUID:      MockUUID,
}

func ConvertContextToString(context *mesh.XRequestContext) string {
	requestContextJson, err := json.Marshal(context)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(requestContextJson)
}
