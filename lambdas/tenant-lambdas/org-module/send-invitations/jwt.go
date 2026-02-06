package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// generateInvitationToken creates a JWT token containing all invitation data
func (svc *Service) GenerateInvitationToken(email, organizationId, organizationName, teamId, role, invitedBy string) (string, error) {
	// Get JWT secret from environment
	secret := os.Getenv("INVITATION_TOKEN_SECRET")
	if secret == "" {
		return "", fmt.Errorf("INVITATION_TOKEN_SECRET environment variable not set")
	}

	// Create claims with invitation data
	now := time.Now()
	claims := InvitationTokenClaims{
		Email:            email,
		OrganizationId:   organizationId,
		OrganizationName: organizationName,
		TeamId:           teamId,
		Role:             role,
		InvitedBy:        invitedBy,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)), // 7 days expiration
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	svc.logger.Printf("Generated invitation token for %s (org: %s, role: %s)", email, organizationId, role)
	return tokenString, nil
}
