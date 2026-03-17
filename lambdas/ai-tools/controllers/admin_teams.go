package controllers

import companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"

// ==================== Admin — Team Functions ====================

// TeamInformation returns metadata for a specific team by teamId.
func (s *Service) TeamInformation(teamId string) (*companylib.TeamMetadata, error) {
	return s.teamsSVC.GetTeamMetadata(teamId)
}

// GetAllOrgTeams returns all teams belonging to an organisation.
func (s *Service) GetAllOrgTeams(orgId string) ([]companylib.TeamMetadata, error) {
	return s.teamsSVC.GetOrganizationTeams(orgId)
}

// GetTeamMembers returns all members of a team.
func (s *Service) GetTeamMembers(teamId string) ([]companylib.TeamMember, error) {
	return s.teamsSVC.GetTeamMembers(teamId)
}

// GetTeamMemberDetail returns the team-specific record for a single member.
func (s *Service) GetTeamMemberDetail(teamId, userName string) (*companylib.TeamMember, error) {
	return s.teamsSVC.GetTeamMemberDetails(teamId, userName)
}

// GetUserTeams returns all teams that a user belongs to.
// Pass the user's Cognito ID for GSI resolution.
func (s *Service) GetUserTeams(userName, cognitoId string) ([]companylib.UserTeamInfo, error) {
	return s.teamsSVC.GetUserTeams(userName, cognitoId)
}

// IsTeamAdmin returns true if the given user is an admin of the specified team.
func (s *Service) IsTeamAdmin(teamId, userName string) (bool, error) {
	return s.teamsSVC.IsTeamAdmin(teamId, userName)
}

// GetTeamAdminCount returns the number of admins in a team.
func (s *Service) GetTeamAdminCount(teamId string) (int, error) {
	return s.teamsSVC.GetAdminCount(teamId)
}
