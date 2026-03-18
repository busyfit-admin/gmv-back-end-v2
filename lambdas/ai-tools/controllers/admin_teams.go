package controllers

import companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"

// ==================== Admin — Team Functions ====================

// TeamInformation returns metadata for a specific team by teamId.
func (s *Service) TeamInformation(teamId string) (*companylib.TeamMetadata, error) {
	s.logger.Printf("ctrl: TeamInformation input: teamId=%q", teamId)
	result, err := s.teamsSVC.GetTeamMetadata(teamId)
	s.logger.Printf("ctrl: TeamInformation output: found=%v err=%v", result != nil, err)
	return result, err
}

// GetAllOrgTeams returns all teams belonging to an organisation.
func (s *Service) GetAllOrgTeams(orgId string) ([]companylib.TeamMetadata, error) {
	s.logger.Printf("ctrl: GetAllOrgTeams input: orgId=%q", orgId)
	result, err := s.teamsSVC.GetOrganizationTeams(orgId)
	s.logger.Printf("ctrl: GetAllOrgTeams output: count=%d err=%v", len(result), err)
	return result, err
}

// GetTeamMembers returns all members of a team.
func (s *Service) GetTeamMembers(teamId string) ([]companylib.TeamMember, error) {
	s.logger.Printf("ctrl: GetTeamMembers input: teamId=%q", teamId)
	result, err := s.teamsSVC.GetTeamMembers(teamId)
	s.logger.Printf("ctrl: GetTeamMembers output: count=%d err=%v", len(result), err)
	return result, err
}

// GetTeamMemberDetail returns the team-specific record for a single member.
func (s *Service) GetTeamMemberDetail(teamId, userName string) (*companylib.TeamMember, error) {
	s.logger.Printf("ctrl: GetTeamMemberDetail input: teamId=%q userName=%q", teamId, userName)
	result, err := s.teamsSVC.GetTeamMemberDetails(teamId, userName)
	s.logger.Printf("ctrl: GetTeamMemberDetail output: found=%v err=%v", result != nil, err)
	return result, err
}

// GetUserTeams returns all teams that a user belongs to.
// Pass the user's Cognito ID for GSI resolution.
func (s *Service) GetUserTeams(userName, cognitoId string) ([]companylib.UserTeamInfo, error) {
	s.logger.Printf("ctrl: GetUserTeams input: userName=%q cognitoId=%q", userName, cognitoId)
	result, err := s.teamsSVC.GetUserTeams(userName, cognitoId)
	s.logger.Printf("ctrl: GetUserTeams output: count=%d err=%v", len(result), err)
	return result, err
}

// IsTeamAdmin returns true if the given user is an admin of the specified team.
func (s *Service) IsTeamAdmin(teamId, userName string) (bool, error) {
	s.logger.Printf("ctrl: IsTeamAdmin input: teamId=%q userName=%q", teamId, userName)
	result, err := s.teamsSVC.IsTeamAdmin(teamId, userName)
	s.logger.Printf("ctrl: IsTeamAdmin output: isAdmin=%v err=%v", result, err)
	return result, err
}

// GetTeamAdminCount returns the number of admins in a team.
func (s *Service) GetTeamAdminCount(teamId string) (int, error) {
	s.logger.Printf("ctrl: GetTeamAdminCount input: teamId=%q", teamId)
	result, err := s.teamsSVC.GetAdminCount(teamId)
	s.logger.Printf("ctrl: GetTeamAdminCount output: count=%d err=%v", result, err)
	return result, err
}
