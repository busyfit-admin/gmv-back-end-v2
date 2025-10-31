package permissions

import (
	"bytes"
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/verifiedpermissions"
	"github.com/aws/aws-sdk-go-v2/service/verifiedpermissions/types"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	"github.com/stretchr/testify/assert"
)

func Test_IsAuthorizedRoleInAdminPortal(t *testing.T) {

	logBuffer := &bytes.Buffer{}
	t.Run("It should allow if the verified permissions has approved the request", func(t *testing.T) {

		avpMockClient := awsclients.MockVerifiedPermissionsClient{
			IsAuthOutput: []verifiedpermissions.IsAuthorizedOutput{
				{
					Decision: types.DecisionAllow,
				},
			},
			IsAuthErrors: []error{
				nil,
			},
		}

		svc := AdminPortalAuthService{
			logger:        log.New(logBuffer, "TEST:", 0),
			avpClient:     &avpMockClient,
			PolicyStoreId: "sample-store-id",
		}

		output := svc.IsAuthorizedRoleInAdminPortal("admin", "GetSubDomains", "ManageSubDomains", "/manageSubDomains")

		assert.Equal(t, true, output)
	})

	t.Run("It should deny if the verified permissions has not approved the request", func(t *testing.T) {

		avpMockClient := awsclients.MockVerifiedPermissionsClient{
			IsAuthOutput: []verifiedpermissions.IsAuthorizedOutput{
				{
					Decision: types.DecisionDeny,
				},
			},
			IsAuthErrors: []error{
				nil,
			},
		}

		svc := AdminPortalAuthService{
			logger:        log.New(logBuffer, "TEST:", 0),
			avpClient:     &avpMockClient,
			PolicyStoreId: "sample-store-id",
		}

		output := svc.IsAuthorizedRoleInAdminPortal("admin", "GetSubDomains", "ManageSubDomains", "/managePortal")

		assert.Equal(t, false, output)
	})
	t.Run("It should deny if the verified permissions has an error", func(t *testing.T) {

		avpMockClient := awsclients.MockVerifiedPermissionsClient{
			IsAuthOutput: []verifiedpermissions.IsAuthorizedOutput{
				{},
			},
			IsAuthErrors: []error{
				fmt.Errorf("error from verified permissions"),
			},
		}

		svc := AdminPortalAuthService{
			logger:        log.New(logBuffer, "TEST:", 0),
			avpClient:     &avpMockClient,
			PolicyStoreId: "sample-store-id",
		}

		output := svc.IsAuthorizedRoleInAdminPortal("admin", "GetSubDomains", "ManageSubDomains", "/ManageSubDomains")

		assert.Equal(t, false, output)
	})
}
