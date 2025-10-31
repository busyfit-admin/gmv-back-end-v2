package Companylib

import (
	"testing"
)

func TestEvaluateRoles(t *testing.T) {
	tests := []struct {
		name       string
		rolesData  map[string]bool
		queryParts []string
		expected   bool
	}{
		{
			name:       "Single role match",
			rolesData:  map[string]bool{"AdminRole": true, "UserManagementRole": false},
			queryParts: []string{"AdminRole"},
			expected:   true,
		},
		{
			name:       "Single role no match",
			rolesData:  map[string]bool{"AdminRole": false, "UserManagementRole": true},
			queryParts: []string{"AdminRole"},
			expected:   false,
		},
		{
			name:       "Multiple roles with AND - all match",
			rolesData:  map[string]bool{"AdminRole": true, "UserManagementRole": true},
			queryParts: []string{"AdminRole", "AND", "UserManagementRole"},
			expected:   true,
		},
		{
			name:       "Multiple roles with AND - one fails",
			rolesData:  map[string]bool{"AdminRole": true, "UserManagementRole": false},
			queryParts: []string{"AdminRole", "AND", "UserManagementRole"},
			expected:   false,
		},
		{
			name:       "Multiple roles with OR - one matches",
			rolesData:  map[string]bool{"AdminRole": true, "UserManagementRole": false},
			queryParts: []string{"AdminRole", "OR", "UserManagementRole"},
			expected:   true,
		},
		{
			name:       "Multiple roles with OR - none match",
			rolesData:  map[string]bool{"AdminRole": false, "UserManagementRole": false},
			queryParts: []string{"AdminRole", "OR", "UserManagementRole"},
			expected:   false,
		},
		{
			name:       "Complex query - mixed operators",
			rolesData:  map[string]bool{"AdminRole": true, "UserManagementRole": false, "AnalyticsRole": true},
			queryParts: []string{"AdminRole", AND, "AnalyticsRole", OR, "UserManagementRole"},
			expected:   true,
		},
		{
			name:       "Empty query",
			rolesData:  map[string]bool{"AdminRole": true},
			queryParts: []string{},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluateRoles(tt.rolesData, tt.queryParts)
			if result != tt.expected {
				t.Errorf("evaluateRoles() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
