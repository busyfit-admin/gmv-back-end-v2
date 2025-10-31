/*

Below are the different states in the tenant lifecycle along with descriptions and suggested parameter names for defining each state in the code:

1. **Initial Onboarding (State: `InitialOnboarding`)**:
   Description: At this stage, basic information about the tenant is collected.
   Parameter Name: `InitialOnboarding`

2. **Initial Onboarding Approval (State: `OnboardingDemo`)**:
   Description: Approval from the Marketing Ops team is obtained to provide an initial application demo.
   Parameter Name: `OnboardingDemo`

3. **Pre-Onboarding Checks (State: `preProvisioningChecks`)**:
   Description: All information is verified for accuracy and approved by the tenant.
   Parameter Name: `preProvisioningChecks`

4. **Trial (State: `Trial`)**:
   Description: Tenants can opt for a trial period, during which a development environment is set up.
   Parameter Name: `Trial`

5. **Trial DEACTIVATED (State: `Deactivated`)**:
   Description: If tenants choose to discontinue the trial, the development environment is inactivated.
   Parameter Name: `Deactivated`

6. **PROVISIONING (State: `PROVISIONING`)**:
   Description: Tenants choose to provision multiple environments including development, user acceptance testing (UAT), and production.
   Parameter Name: `PROVISIONING`

7. **Active (State: `Active`)**:
   Description: Tenants transition to the active state after completing the trial period and opting for the paid version.
   Parameter Name: `Active`

These parameter names can be used to maintain the state of each tenant in the code, enabling easy tracking and management of the tenant lifecycle within the SaaS-based Loyalty solution.
*/

/*
Expected Tenant Details API response:

	{
		"TenantId" : "abc-123456", // Top level information to be received from TenantsDetails Table
		"Status" : "",
		"TenantDetails" : {
			"TenantName" : "abc abc ",
			"TenantPrimaryContact" : "+877234324324",
			"TenantPrimaryEmail" : "email@email.com",
			"TenantSecondaryContact" : "+877234324324",
			"TenantSecondaryEmail" : "email@email.com",
			"TenantAddress" : "address "
		},
		"Billing" : {},
		"Environments" : { // Data to be received from Subdomains DDB table
			"EnvId": {
				"EnvType": "UAT",
				"EnvName": "UAT",
				"ProvisionStartDate" : "Date",
				"Domain": "domain.com",
				"Status" : "Active" // Can be either Active / Inactive
			},
			"EnvId": {
				"EnvType": "DEV",
				"EnvName": "DEV",
				"ProvisionStartDate" : "Date",
				"Domain": "domain.com",
				"Status" : "Active" // Can be either Active / Inactive
			}
		},
		"Stages" : { // Data to be received from Tenants stages DDB Table
			"InitialOnboarding" : {
				"OverallStatus" : "Completed",
				"FollowUpDetails" : {
					"commentId1" : {
						"comment" : "abc xyz",
						"Status" : "FollowUp1"
					},
					"commentId2" : {
						"comment" : "abc xyz",
						"Status" : "FollowUp1"
					}
				}
			},
			"OnboardingDemo" : {
				"OverallStatus" : "Completed",
				"FollowUpDetails" : {
					"commentId1" : {
						"comment" : "abc xyz",
						"Status" : "FollowUp1"
					},
					"commentId2" : {
						"comment" : "abc xyz",
						"Status" : "FollowUp1"
					}
				}
			}
		}

	}
*/
package adminlib

import "errors"

// Stage Status Types
const (
	UNDEFINED = "UNDEFINED" // No status has been set yet for a certain stage. This can happen when no comments or Tenant has not been processed to this stage

	IN_PROG = "IN_PROG" // Currently the stage is in progress.

	IN_PROG_TIME_EXCEEDED = "IN_PROG_TIME_EXCEEDED" // In Prog time has exceeded

	FOLLOWUP = "FOLLOWUP" // FOLLOW UP in progress.

	FOLLOWUP_TIME_EXCEEDED = "FOLLOWUP_TIME_EXCEEDED" // Follow up timeline has exceeded for that stage. Needs attention

	ESCALATED = "ESCALATED" // The stage is currently ESCALATED

	COMPLETED = "COMPLETED" // The stage work has been addressed and acknowledged from the Tenants

)

// Stage Types
const (
	INITIAL_ONBOARDING          = "InitialOnboarding" // Tenants have registered via the LS Website or Marketing team has logged a new Tenant Request
	INITIAL_ONBOARDING_DESC     = ""
	INITIAL_ONBOARDING_STAGE_ID = "STG01"

	ONBOARDING_DEMO          = "OnboardingDemo" // Approved for Demo of the LS Website
	ONBOARDING_DEMO_DESC     = ""
	ONBOARDING_DEMO_STAGE_ID = "STG02"

	TRIAL_SETUP          = "TrialSetup" // Setup the Trail env
	TRIAL_SETUP_DESC     = ""
	TRIAL_SETUP_STAGE_ID = "STG03"

	TRIAL_IN_PROG          = "TrialInProg" // Trail setup after Demo ( usually for 30 days )
	TRIAL_IN_PROG_DESC     = ""
	TRIAL_IN_PROG_STAGE_ID = "STG04"

	TRAIL_DISCONTINUED          = "Trail_Discontinued" // Trail Discontinued but not yet Provisioned.
	TRAIL_DISCONTINUED_DESC     = ""
	TRAIL_DISCONTINUED_STAGE_ID = "STG05"

	PRE_PROVISIONING_CHECKS          = "PreProvisioningChecks" // Pre checks for the Tenants information like emailId etc
	PRE_PROVISIONING_CHECKS_DESC     = ""
	PRE_PROVISIONING_CHECKS_STAGE_ID = "STG06"

	PROVISIONING          = "Provisioning" // Provisioning the active env for the Tenants
	PROVISIONING_DESC     = ""
	PROVISIONING_STAGE_ID = "STG07"

	ACTIVE          = "Active" // Tenant is currently active
	ACTIVE_DESC     = ""
	ACTIVE_STAGE_ID = "STG08"

	INACTIVE          = "Inactive" // Tenant has moved to Inactive after Being Active. The ENV will be in Inactive state for a defined period of time
	INACTIVE_DESC     = ""
	INACTIVE_STAGE_ID = "STG09"

	DEACTIVATED          = "Deactivated" // Tenant has formally agreed to not proceed with the Services
	DEACTIVATED_DESC     = ""
	DEACTIVATED_STAGE_ID = "STG10"
)

// Function to verify if the next step in tenant lifecycle is valid
func IsNextStepValid(currentStep, nextStep string) error {
	switch currentStep {
	case INITIAL_ONBOARDING:
		if nextStep != ONBOARDING_DEMO {
			return errors.New("next step should be OnboardingDemo")
		}
	case ONBOARDING_DEMO:
		if nextStep != PRE_PROVISIONING_CHECKS {
			return errors.New("next step should be preProvisioningChecks")
		}
	case PRE_PROVISIONING_CHECKS:
		if nextStep != TRIAL_IN_PROG && nextStep != PROVISIONING {
			return errors.New("next step should be Trial or PROVISIONING")
		}
	case TRIAL_IN_PROG:
		if nextStep != DEACTIVATED && nextStep != ACTIVE {
			return errors.New("next step should be Deactivated or Active")
		}
	case PROVISIONING:
		if nextStep != ACTIVE {
			return errors.New("next step should be Active")
		}
	case ACTIVE:
		return errors.New("tenant is already in Active state, no further steps")
	default:
		return errors.New("invalid current step")
	}
	return nil
}
