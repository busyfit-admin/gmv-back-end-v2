package controllers

import companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"

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
func (s *Service) FindEmployeeByCognitoId(cognitoId string) (companylib.EmployeeDynamodbData, error) {
	return s.empSVC.GetEmployeeDataByCognitoId(cognitoId)
}

// CheckEmployeeExists returns true if an employee with the given userName exists.
func (s *Service) CheckEmployeeExists(userName string) (bool, error) {
	return s.empSVC.CheckUserExists(userName)
}

// GetAllEmployeeGroups returns all employee groups mapped by group name.
func (s *Service) GetAllEmployeeGroups() (map[string]companylib.EmployeeGroups, error) {
	return s.empSVC.GetAllEmployeeGroupsInMap()
}
