package main

import (
	"testing"

	adminlib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/admin-lib"
	"github.com/stretchr/testify/assert"
)

func Test_MapStageDataToResponse(t *testing.T) {

	inputData1 := []adminlib.TenantStages{
		{
			TenantId:    "tenantId-1",
			StageId:     "STG01",
			StageStatus: "INPROG",
			StageComments: []adminlib.CommentData{
				{
					Comment:         "This is a test comment",
					CommentBy:       "Person1",
					UpdateTimeStamp: "2024-04-01",
				},
				{
					Comment:         "This is a test comment 2",
					CommentBy:       "Person2",
					UpdateTimeStamp: "2024-04-02",
				},
			},
		},
		{
			TenantId:    "tenantId-1",
			StageId:     "STG04",
			StageStatus: "COMPLETED",
			StageComments: []adminlib.CommentData{
				{
					Comment:         "This is a test comment",
					CommentBy:       "Person1",
					UpdateTimeStamp: "2024-04-01",
				},
				{
					Comment:         "This is a test comment 2",
					CommentBy:       "Person2",
					UpdateTimeStamp: "2024-04-02",
				},
			},
		},
	}
	t.Run("It should take the Stages input and return the required API response struct", func(t *testing.T) {

		output := MapStageDataToResponse(inputData1)

		expectedResp := GetTenantStagesResponse{
			InitialOnboarding: StageData{
				OverallStatus: "INPROG",
				FollowUpDetails: []adminlib.CommentData{
					{
						Comment:         "This is a test comment",
						CommentBy:       "Person1",
						UpdateTimeStamp: "2024-04-01",
					},
					{
						Comment:         "This is a test comment 2",
						CommentBy:       "Person2",
						UpdateTimeStamp: "2024-04-02",
					},
				},
			},
			TrialInProg: StageData{
				OverallStatus: "COMPLETED",
				FollowUpDetails: []adminlib.CommentData{
					{
						Comment:         "This is a test comment",
						CommentBy:       "Person1",
						UpdateTimeStamp: "2024-04-01",
					},
					{
						Comment:         "This is a test comment 2",
						CommentBy:       "Person2",
						UpdateTimeStamp: "2024-04-02",
					},
				},
			},
		}

		assert.Equal(t, expectedResp, output)

	})

	t.Run("It should take the Empty Stages input and return the required API response struct", func(t *testing.T) {

		output := MapStageDataToResponse([]adminlib.TenantStages{})

		expectedResp := GetTenantStagesResponse{}

		assert.Equal(t, expectedResp, output)

	})

}
