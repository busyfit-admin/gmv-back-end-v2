package controllers

import companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"

// ==================== Admin — Organisation Functions ====================

// GetOrgInfo returns the full organisation record for the given organisationId.
func (s *Service) GetOrgInfo(orgId string) (*companylib.Organization, error) {
	return s.orgSVC.GetOrganization(orgId)
}

// GetOrgAdmins returns all admins of an organisation.
// requestingUser is the userName of the caller (used for access checks inside the service).
func (s *Service) GetOrgAdmins(orgId, requestingUser string) ([]companylib.OrgAdmin, error) {
	return s.orgSVC.GetOrgAdmins(orgId, requestingUser)
}

// GetOrgUsers returns all members (employees) registered under an organisation.
func (s *Service) GetOrgUsers(orgId string) ([]companylib.OrgMember, error) {
	return s.orgSVC.GetOrgUsers(orgId)
}

// IsOrgAdmin returns true if the given user is an admin of the specified organisation.
func (s *Service) IsOrgAdmin(orgId, userName string) (bool, error) {
	return s.orgSVC.IsOrgAdmin(orgId, userName)
}

// GetAdminOrganizations returns the list of organisations where the given user is an admin.
func (s *Service) GetAdminOrganizations(userName string) ([]companylib.Organization, error) {
	return s.orgSVC.GetAdminsOrganizations(userName)
}
