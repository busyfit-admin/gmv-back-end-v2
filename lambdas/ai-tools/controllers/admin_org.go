package controllers

import companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"

// ==================== Admin — Organisation Functions ====================

// GetOrgInfo returns the full organisation record for the given organisationId.
func (s *Service) GetOrgInfo(orgId string) (*companylib.Organization, error) {
	s.logger.Printf("ctrl: GetOrgInfo input: orgId=%q", orgId)
	result, err := s.orgSVC.GetOrganization(orgId)
	s.logger.Printf("ctrl: GetOrgInfo output: found=%v err=%v", result != nil, err)
	return result, err
}

// GetOrgAdmins returns all admins of an organisation.
// requestingUser is the userName of the caller (used for access checks inside the service).
func (s *Service) GetOrgAdmins(orgId, requestingUser string) ([]companylib.OrgAdmin, error) {
	s.logger.Printf("ctrl: GetOrgAdmins input: orgId=%q requestingUser=%q", orgId, requestingUser)
	result, err := s.orgSVC.GetOrgAdmins(orgId, requestingUser)
	s.logger.Printf("ctrl: GetOrgAdmins output: count=%d err=%v", len(result), err)
	return result, err
}

// GetOrgUsers returns all members (employees) registered under an organisation.
func (s *Service) GetOrgUsers(orgId string) ([]companylib.OrgMember, error) {
	s.logger.Printf("ctrl: GetOrgUsers input: orgId=%q", orgId)
	result, err := s.orgSVC.GetOrgUsers(orgId)
	s.logger.Printf("ctrl: GetOrgUsers output: count=%d err=%v", len(result), err)
	return result, err
}

// IsOrgAdmin returns true if the given user is an admin of the specified organisation.
func (s *Service) IsOrgAdmin(orgId, userName string) (bool, error) {
	s.logger.Printf("ctrl: IsOrgAdmin input: orgId=%q userName=%q", orgId, userName)
	result, err := s.orgSVC.IsOrgAdmin(orgId, userName)
	s.logger.Printf("ctrl: IsOrgAdmin output: isAdmin=%v err=%v", result, err)
	return result, err
}

// GetAdminOrganizations returns the list of organisations where the given user is an admin.
func (s *Service) GetAdminOrganizations(userName string) ([]companylib.Organization, error) {
	s.logger.Printf("ctrl: GetAdminOrganizations input: userName=%q", userName)
	result, err := s.orgSVC.GetAdminsOrganizations(userName)
	s.logger.Printf("ctrl: GetAdminOrganizations output: count=%d err=%v", len(result), err)
	return result, err
}

// GetUserOrgID returns the organisation ID that the given user belongs to.
// Returns an empty string (and no error) when no membership record is found.
func (s *Service) GetUserOrgID(userName string) string {
	s.logger.Printf("ctrl: GetUserOrgID input: userName=%q", userName)
	membership, err := s.orgSVC.GetUserOrganizationMembership(userName)
	if err != nil || membership == nil {
		s.logger.Printf("ctrl: GetUserOrgID output: orgId=%q (not found, err=%v)", "", err)
		return ""
	}
	s.logger.Printf("ctrl: GetUserOrgID output: orgId=%q", membership.OrganizationId)
	return membership.OrganizationId
}
