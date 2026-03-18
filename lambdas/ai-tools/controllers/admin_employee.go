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
	s.logger.Printf("ctrl: EmployeeInformation input: userName=%q", userName)
	result, err := s.empSVC.GetEmployeeDataByUserName(userName)
	s.logger.Printf("ctrl: EmployeeInformation output: userName=%q err=%v", result.UserName, err)
	return result, err
}

// GetEmployeeBasicData returns a lightweight summary (name, email, role) for an employee.
func (s *Service) GetEmployeeBasicData(userName string) (companylib.GetBasicEmployeeData, error) {
	s.logger.Printf("ctrl: GetEmployeeBasicData input: userName=%q", userName)
	result, err := s.empSVC.GetEmployeeDataByUserNameBasicData(userName)
	s.logger.Printf("ctrl: GetEmployeeBasicData output: err=%v", err)
	return result, err
}

// GetAllEmployees returns every employee record in the organisation.
func (s *Service) GetAllEmployees() ([]companylib.EmployeeDynamodbData, error) {
	s.logger.Printf("ctrl: GetAllEmployees input: (no params)")
	result, err := s.empSVC.GetAllEmployeeData()
	s.logger.Printf("ctrl: GetAllEmployees output: count=%d err=%v", len(result), err)
	return result, err
}

// FindEmployeeByEmail looks up an employee by their email address.
func (s *Service) FindEmployeeByEmail(email string) (companylib.EmployeeDynamodbData, error) {
	s.logger.Printf("ctrl: FindEmployeeByEmail input: email=%q", email)
	result, err := s.empSVC.GetEmployeeDataByEmail(email)
	s.logger.Printf("ctrl: FindEmployeeByEmail output: userName=%q err=%v", result.UserName, err)
	return result, err
}

// FindEmployeeByCognitoId looks up an employee by their Cognito sub/user ID.
// ctx must be the per-request context so that X-Ray subsegments attach correctly.
func (s *Service) FindEmployeeByCognitoId(ctx context.Context, cognitoId string) (companylib.EmployeeDynamodbData, error) {
	s.logger.Printf("ctrl: FindEmployeeByCognitoId input: cognitoId=%q", cognitoId)
	out, err := s.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.empSVC.EmployeeTable),
		IndexName:              aws.String(s.empSVC.EmployeeTable_CognitoId_Index),
		KeyConditionExpression: aws.String("CognitoId = :CognitoId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":CognitoId": &types.AttributeValueMemberS{Value: cognitoId},
		},
	})
	if err != nil {
		s.logger.Printf("ctrl: FindEmployeeByCognitoId output: err=%v", err)
		return companylib.EmployeeDynamodbData{}, err
	}
	if out.Count == 0 {
		s.logger.Printf("ctrl: FindEmployeeByCognitoId output: not found")
		return companylib.EmployeeDynamodbData{}, fmt.Errorf("no employee found for cognito-id: %s", cognitoId)
	}
	var emp companylib.EmployeeDynamodbData
	if err := attributevalue.UnmarshalMap(out.Items[0], &emp); err != nil {
		s.logger.Printf("ctrl: FindEmployeeByCognitoId output: unmarshal err=%v", err)
		return companylib.EmployeeDynamodbData{}, err
	}
	if emp.ProfilePic == "" {
		emp.ProfilePic = "default.jpg"
	}
	s.logger.Printf("ctrl: FindEmployeeByCognitoId output: userName=%q", emp.UserName)
	return emp, nil
}

// CheckEmployeeExists returns true if an employee with the given userName exists.
func (s *Service) CheckEmployeeExists(userName string) (bool, error) {
	s.logger.Printf("ctrl: CheckEmployeeExists input: userName=%q", userName)
	result, err := s.empSVC.CheckUserExists(userName)
	s.logger.Printf("ctrl: CheckEmployeeExists output: exists=%v err=%v", result, err)
	return result, err
}

// GetAllEmployeeGroups returns all employee groups mapped by group name.
func (s *Service) GetAllEmployeeGroups() (map[string]companylib.EmployeeGroups, error) {
	s.logger.Printf("ctrl: GetAllEmployeeGroups input: (no params)")
	result, err := s.empSVC.GetAllEmployeeGroupsInMap()
	s.logger.Printf("ctrl: GetAllEmployeeGroups output: count=%d err=%v", len(result), err)
	return result, err
}
