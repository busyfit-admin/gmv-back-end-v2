package controllers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
)

// ==================== Admin — Employee Functions ====================

// EmployeeInformation returns the full DynamoDB record for an employee looked up by userName.
func (s *Service) EmployeeInformation(userName string) (companylib.EmployeeDynamodbData, error) {
	return s.empSVC.GetEmployeeDataByUserName(userName)
}

// GetEmployeeBasicData returns a lightweight summary (name, email, role) for an employee.
func (s *Service) GetEmployeeBasicData(userName string) (companylib.GetBasicEmployeeData, error) {
	return s.empSVC.GetEmployeeDataByUserNameBasicData(userName)
}

// GetAllEmployees returns every employee record in the organisation.
func (s *Service) GetAllEmployees() ([]companylib.EmployeeDynamodbData, error) {
	return s.empSVC.GetAllEmployeeData()
}

// FindEmployeeByEmail looks up an employee by their email address.
func (s *Service) FindEmployeeByEmail(email string) (companylib.EmployeeDynamodbData, error) {
	return s.empSVC.GetEmployeeDataByEmail(email)
}

// FindEmployeeByCognitoId looks up an employee by their Cognito sub/user ID.
// ctx must be the per-request context so that X-Ray subsegments attach correctly.
func (s *Service) FindEmployeeByCognitoId(ctx context.Context, cognitoId string) (companylib.EmployeeDynamodbData, error) {
	out, err := s.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.empSVC.EmployeeTable),
		IndexName:              aws.String(s.empSVC.EmployeeTable_CognitoId_Index),
		KeyConditionExpression: aws.String("CognitoId = :CognitoId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":CognitoId": &types.AttributeValueMemberS{Value: cognitoId},
		},
	})
	if err != nil {
		return companylib.EmployeeDynamodbData{}, err
	}
	if out.Count == 0 {
		return companylib.EmployeeDynamodbData{}, fmt.Errorf("no employee found for cognito-id: %s", cognitoId)
	}
	var emp companylib.EmployeeDynamodbData
	if err := attributevalue.UnmarshalMap(out.Items[0], &emp); err != nil {
		return companylib.EmployeeDynamodbData{}, err
	}
	if emp.ProfilePic == "" {
		emp.ProfilePic = "default.jpg"
	}
	return emp, nil
}

// CheckEmployeeExists returns true if an employee with the given userName exists.
func (s *Service) CheckEmployeeExists(userName string) (bool, error) {
	return s.empSVC.CheckUserExists(userName)
}

// GetAllEmployeeGroups returns all employee groups mapped by group name.
func (s *Service) GetAllEmployeeGroups() (map[string]companylib.EmployeeGroups, error) {
	return s.empSVC.GetAllEmployeeGroupsInMap()
}
