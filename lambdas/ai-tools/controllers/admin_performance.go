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
	f := map[string]string{}
	if filters.Status != "" {
		f["status"] = filters.Status
	}
	if filters.Year != "" {
		f["year"] = filters.Year
	}
	return s.perfSVC.ListPerformanceCycles(orgID, f, companylib.ListQueryOptions{}, includeQuarters, includeKPIs, includeOKRs)
}

// GetPerformanceCycleDetails returns the full detail of a single performance cycle.
func (s *Service) GetPerformanceCycleDetails(cycleID string, includeQuarters, includeKPIs, includeOKRs, includeAnalytics bool) (map[string]interface{}, error) {
	return s.perfSVC.GetPerformanceCycleDetails(cycleID, includeQuarters, includeKPIs, includeOKRs, includeAnalytics)
}

// GetCycleAnalytics returns aggregated progress analytics for a performance cycle.
func (s *Service) GetCycleAnalytics(cycleID string) (map[string]interface{}, error) {
	return s.perfSVC.GetCycleAnalytics(cycleID)
}

// ==================== Quarters ====================

// GetAllQuarters lists all quarters within a performance cycle.
func (s *Service) GetAllQuarters(cycleID string) (map[string]interface{}, error) {
	return s.perfSVC.ListQuarters(cycleID)
}

// GetQuarterDetails returns the full detail of a single quarter.
// Set the include flags to preload KPIs, OKRs, meeting notes, or pending reviews.
func (s *Service) GetQuarterDetails(quarterID string, includeKPIs, includeOKRs, includeMeetingNotes, includePendingReviews bool) (map[string]interface{}, error) {
	return s.perfSVC.GetQuarterDetails(quarterID, includeKPIs, includeOKRs, includeMeetingNotes, includePendingReviews)
}

// GetQuarterAnalytics returns aggregated progress analytics for a quarter.
func (s *Service) GetQuarterAnalytics(quarterID string) (map[string]interface{}, error) {
	return s.perfSVC.GetQuarterAnalytics(quarterID)
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
	return s.perfSVC.ListKPIs(orgID, f, companylib.ListQueryOptions{}, includeSubKPIs)
}

// GetKPIDetail returns the full detail (including optional sub-KPIs and value history) for one KPI.
func (s *Service) GetKPIDetail(kpiID string, includeSubKPIs, includeValueHistory bool) (map[string]interface{}, error) {
	return s.perfSVC.GetKPIDetails(kpiID, includeSubKPIs, includeValueHistory)
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
	return s.perfSVC.ListOKRs(orgID, f, companylib.ListQueryOptions{}, includeKeyResults)
}

// GetOKRDetail returns the full detail of a single OKR.
func (s *Service) GetOKRDetail(okrID string, includeKeyResults, includeProgressHistory bool) (map[string]interface{}, error) {
	return s.perfSVC.GetOKRDetails(okrID, includeKeyResults, includeProgressHistory)
}

// ==================== Org-Level Goals ====================

// GetOrgGoalDetail returns the full detail of a single org-level goal (KPI/OKR).
// Set the include flags to load related data: value history, tagged teams, sub-items,
// ladder-up chain, and private tasks for a specific user.
func (s *Service) GetOrgGoalDetail(goalID string, includeValueHistory, includeTaggedTeams, includeSubItems, includeLadderUp bool, userName string) (map[string]interface{}, error) {
	return s.perfSVC.GetGoalDetails(goalID, includeValueHistory, includeTaggedTeams, includeSubItems, includeLadderUp, false, userName)
}

// GetTeamOrgGoals returns all org-level goals tagged to a team, optionally filtered by type.
// goalType: "kpi" | "okr" | "" (all).
func (s *Service) GetTeamOrgGoals(teamID, orgID, goalType string) (map[string]interface{}, error) {
	return s.perfSVC.GetTeamGoals(teamID, orgID, goalType, map[string]string{}, companylib.ListQueryOptions{})
}

// GetOrgGoalSubItems returns the direct sub-items (child goals) of an org-level goal.
func (s *Service) GetOrgGoalSubItems(goalID string) (map[string]interface{}, error) {
	return s.perfSVC.GetGoalSubItems(goalID)
}

// ==================== Quarter Meeting Notes ====================

// GetQuarterMeetingNotes lists all meeting notes for a quarter.
// sortBy: "date"; order: "asc" | "desc".
func (s *Service) GetQuarterMeetingNotes(quarterID, sortBy, order string) (map[string]interface{}, error) {
	return s.perfSVC.ListMeetingNotes(quarterID, sortBy, order)
}
