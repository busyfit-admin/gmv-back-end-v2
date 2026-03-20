package controllers

import companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"

// ==================== Admin — Org Performance Functions ====================
//
// All functions in this file wrap PerformanceService from company-lib.
// Return values are map[string]interface{} because the underlying service
// returns rich, nested payloads built via DynamoDB projections.

// PerformanceCycleFilters are the optional filters accepted by ListPerformanceCycles.
// All fields correspond to PerformanceService filter keys.
type PerformanceCycleFilters struct {
	Status string // "active" | "archived" | "completed"
	Year   string // e.g. "2024"
}

// ==================== Performance Cycles ====================

// GetPerformanceCycles lists all performance cycles for an organisation.
// Set includeQuarters/includeKPIs/includeOKRs to true to preload those child records.
func (s *Service) GetPerformanceCycles(orgID string, filters PerformanceCycleFilters, includeQuarters, includeKPIs, includeOKRs bool) (map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetPerformanceCycles input: orgID=%q status=%q includeQuarters=%v includeKPIs=%v includeOKRs=%v", orgID, filters.Status, includeQuarters, includeKPIs, includeOKRs)
	f := map[string]string{}
	if filters.Status != "" {
		f["status"] = filters.Status
	}
	if filters.Year != "" {
		f["year"] = filters.Year
	}
	result, err := s.perfSVC.ListPerformanceCycles(orgID, f, companylib.ListQueryOptions{}, includeQuarters, includeKPIs, includeOKRs)
	s.logger.Printf("ctrl: GetPerformanceCycles output: err=%v", err)
	return result, err
}

// GetPerformanceCycleDetails returns the full detail of a single performance cycle.
func (s *Service) GetPerformanceCycleDetails(cycleID string, includeQuarters, includeKPIs, includeOKRs, includeAnalytics bool) (map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetPerformanceCycleDetails input: cycleID=%q includeQuarters=%v includeKPIs=%v includeOKRs=%v includeAnalytics=%v", cycleID, includeQuarters, includeKPIs, includeOKRs, includeAnalytics)
	result, err := s.perfSVC.GetPerformanceCycleDetails(cycleID, includeQuarters, includeKPIs, includeOKRs, includeAnalytics)
	s.logger.Printf("ctrl: GetPerformanceCycleDetails output: err=%v", err)
	return result, err
}

// GetCycleAnalytics returns aggregated progress analytics for a performance cycle.
func (s *Service) GetCycleAnalytics(cycleID string) (map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetCycleAnalytics input: cycleID=%q", cycleID)
	result, err := s.perfSVC.GetCycleAnalytics(cycleID)
	s.logger.Printf("ctrl: GetCycleAnalytics output: err=%v", err)
	return result, err
}

// ==================== Quarters ====================

// GetAllQuarters lists all quarters within a performance cycle.
func (s *Service) GetAllQuarters(cycleID string) (map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetAllQuarters input: cycleID=%q", cycleID)
	result, err := s.perfSVC.ListQuarters(cycleID)
	s.logger.Printf("ctrl: GetAllQuarters output: err=%v", err)
	return result, err
}

// GetQuarterDetails returns the full detail of a single quarter.
// Set the include flags to preload KPIs, OKRs, meeting notes, or pending reviews.
func (s *Service) GetQuarterDetails(quarterID string, includeKPIs, includeOKRs, includeMeetingNotes, includePendingReviews bool) (map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetQuarterDetails input: quarterID=%q includeKPIs=%v includeOKRs=%v", quarterID, includeKPIs, includeOKRs)
	result, err := s.perfSVC.GetQuarterDetails(quarterID, includeKPIs, includeOKRs, includeMeetingNotes, includePendingReviews)
	s.logger.Printf("ctrl: GetQuarterDetails output: err=%v", err)
	return result, err
}

// GetQuarterAnalytics returns aggregated progress analytics for a quarter.
func (s *Service) GetQuarterAnalytics(quarterID string) (map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetQuarterAnalytics input: quarterID=%q", quarterID)
	result, err := s.perfSVC.GetQuarterAnalytics(quarterID)
	s.logger.Printf("ctrl: GetQuarterAnalytics output: err=%v", err)
	return result, err
}

// ==================== KPIs ====================

// KPIFilters are the optional filters accepted by GetAllKPIs.
type KPIFilters struct {
	Status    string // "active" | "archived" | "completed"
	CycleID   string
	QuarterID string
}

// GetAllKPIs lists all KPIs for an organisation. Set includeSubKPIs to true to
// preload child KPIs in the same response.
func (s *Service) GetAllKPIs(orgID string, filters KPIFilters, includeSubKPIs bool) (map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetAllKPIs input: orgID=%q status=%q cycleID=%q includeSubKPIs=%v", orgID, filters.Status, filters.CycleID, includeSubKPIs)
	f := map[string]string{}
	if filters.Status != "" {
		f["status"] = filters.Status
	}
	if filters.CycleID != "" {
		f["cycleId"] = filters.CycleID
	}
	if filters.QuarterID != "" {
		f["quarterId"] = filters.QuarterID
	}
	result, err := s.perfSVC.ListKPIs(orgID, f, companylib.ListQueryOptions{}, includeSubKPIs)
	s.logger.Printf("ctrl: GetAllKPIs output: err=%v", err)
	return result, err
}

// GetKPIDetail returns the full detail (including optional sub-KPIs and value history) for one KPI.
func (s *Service) GetKPIDetail(kpiID string, includeSubKPIs, includeValueHistory bool) (map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetKPIDetail input: kpiID=%q includeSubKPIs=%v includeValueHistory=%v", kpiID, includeSubKPIs, includeValueHistory)
	result, err := s.perfSVC.GetKPIDetails(kpiID, includeSubKPIs, includeValueHistory)
	s.logger.Printf("ctrl: GetKPIDetail output: err=%v", err)
	return result, err
}

// ==================== OKRs ====================

// OKRFilters are the optional filters accepted by GetAllOKRs.
type OKRFilters struct {
	Status    string // "active" | "archived" | "completed"
	CycleID   string
	QuarterID string
}

// GetAllOKRs lists all OKRs for an organisation. Set includeKeyResults to true to
// preload key results in the same response.
func (s *Service) GetAllOKRs(orgID string, filters OKRFilters, includeKeyResults bool) (map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetAllOKRs input: orgID=%q status=%q cycleID=%q includeKeyResults=%v", orgID, filters.Status, filters.CycleID, includeKeyResults)
	f := map[string]string{}
	if filters.Status != "" {
		f["status"] = filters.Status
	}
	if filters.CycleID != "" {
		f["cycleId"] = filters.CycleID
	}
	if filters.QuarterID != "" {
		f["quarterId"] = filters.QuarterID
	}
	result, err := s.perfSVC.ListOKRs(orgID, f, companylib.ListQueryOptions{}, includeKeyResults)
	s.logger.Printf("ctrl: GetAllOKRs output: err=%v", err)
	return result, err
}

// GetOKRDetail returns the full detail of a single OKR.
func (s *Service) GetOKRDetail(okrID string, includeKeyResults, includeProgressHistory bool) (map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetOKRDetail input: okrID=%q includeKeyResults=%v includeProgressHistory=%v", okrID, includeKeyResults, includeProgressHistory)
	result, err := s.perfSVC.GetOKRDetails(okrID, includeKeyResults, includeProgressHistory)
	s.logger.Printf("ctrl: GetOKRDetail output: err=%v", err)
	return result, err
}

// ==================== Org-Level Goals ====================

// GetOrgGoalDetail returns the full detail of a single org-level goal (KPI/OKR).
// Set the include flags to load related data: value history, tagged teams, sub-items,
// ladder-up chain, and private tasks for a specific user.
func (s *Service) GetOrgGoalDetail(goalID string, includeValueHistory, includeTaggedTeams, includeSubItems, includeLadderUp bool, userName string) (map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetOrgGoalDetail input: goalID=%q userName=%q includeValueHistory=%v includeTaggedTeams=%v includeSubItems=%v includeLadderUp=%v", goalID, userName, includeValueHistory, includeTaggedTeams, includeSubItems, includeLadderUp)
	result, err := s.perfSVC.GetGoalDetails(goalID, includeValueHistory, includeTaggedTeams, includeSubItems, includeLadderUp, false, userName)
	s.logger.Printf("ctrl: GetOrgGoalDetail output: err=%v", err)
	return result, err
}

// GetTeamOrgGoals returns all org-level goals tagged to a team, optionally filtered by type.
// goalType: "kpi" | "okr" | "" (all).
func (s *Service) GetTeamOrgGoals(teamID, orgID, goalType string) (map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetTeamOrgGoals input: teamID=%q orgID=%q goalType=%q", teamID, orgID, goalType)
	result, err := s.perfSVC.GetTeamGoals(teamID, orgID, goalType, map[string]string{}, companylib.ListQueryOptions{})
	s.logger.Printf("ctrl: GetTeamOrgGoals output: err=%v", err)
	return result, err
}

// GetOrgGoalSubItems returns the direct sub-items (child goals) of an org-level goal.
func (s *Service) GetOrgGoalSubItems(goalID string) (map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetOrgGoalSubItems input: goalID=%q", goalID)
	result, err := s.perfSVC.GetGoalSubItems(goalID)
	s.logger.Printf("ctrl: GetOrgGoalSubItems output: err=%v", err)
	return result, err
}

// ==================== Quarter Meeting Notes ====================

// GetQuarterMeetingNotes lists all meeting notes for a quarter.
// sortBy: "date"; order: "asc" | "desc".
func (s *Service) GetQuarterMeetingNotes(quarterID, sortBy, order string) (map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetQuarterMeetingNotes input: quarterID=%q sortBy=%q order=%q", quarterID, sortBy, order)
	result, err := s.perfSVC.ListMeetingNotes(quarterID, sortBy, order)
	s.logger.Printf("ctrl: GetQuarterMeetingNotes output: err=%v", err)
	return result, err
}
