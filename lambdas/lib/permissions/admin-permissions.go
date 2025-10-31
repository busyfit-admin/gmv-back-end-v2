package permissions

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/verifiedpermissions"
	"github.com/aws/aws-sdk-go-v2/service/verifiedpermissions/types"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
)

type AdminPortalAuthService struct {
	ctx    context.Context
	logger *log.Logger

	avpClient     awsclients.VerifiedPermissionsClient
	PolicyStoreId string
}

func CreateAdminPortalAuthService(ctx context.Context, logger *log.Logger, avpClient awsclients.VerifiedPermissionsClient) AdminPortalAuthService {

	return AdminPortalAuthService{
		ctx:       ctx,
		logger:    logger,
		avpClient: avpClient,
	}
}

// Checks Role Auth for Admin Portal
/*
{
    "principal": {
        "entityType": "PhotoFlash::User",
        "entityId": "alice"
    },
    "action": {
        "actionType": "Action",
        "actionId": "view"
    },
    "resource": {
        "entityType": "PhotoFlash::Photo",
        "entityId": "VacationPhoto94.jpg"
    },
    "policyStoreId": "PSEXAMPLEabcdefg111111"
}

*/
func (svc *AdminPortalAuthService) IsAuthorizedRoleInAdminPortal(who string, action string, forResource string, endpoint string) bool {

	output, err := svc.avpClient.IsAuthorized(svc.ctx, &verifiedpermissions.IsAuthorizedInput{
		PolicyStoreId: aws.String(svc.PolicyStoreId),

		Action: &types.ActionIdentifier{
			ActionId:   aws.String("Action"), // Default Value
			ActionType: aws.String(action),
		},
		Principal: &types.EntityIdentifier{
			EntityId:   aws.String("AdminPortal::Role"), // Default Value
			EntityType: aws.String(who),
		},
		Resource: &types.EntityIdentifier{
			EntityId:   aws.String("AdminPortal::" + forResource),
			EntityType: aws.String(endpoint),
		},
	})

	if err != nil {
		svc.logger.Printf("Failed to authorize with auth error: %v", err)
		return false
	}

	if output.Decision == types.DecisionAllow {
		return true
	}

	return false
}
