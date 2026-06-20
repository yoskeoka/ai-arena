package service

import (
	"context"
	"fmt"
	"strings"

	oidc "github.com/coreos/go-oidc/v3/oidc"
)

// OIDCVerifiedClaims is the subset of an OIDC verified token needed for normalized identity extraction.
type OIDCVerifiedClaims interface {
	Claims(any) error
}

// OIDCIdentityVerifier verifies a raw ID token and exposes its claims.
type OIDCIdentityVerifier interface {
	Verify(ctx context.Context, rawIDToken string) (OIDCVerifiedClaims, error)
}

// CoreOIDCIdentityVerifier adapts go-oidc's verifier to the repo-local seam.
type CoreOIDCIdentityVerifier struct {
	verifier *oidc.IDTokenVerifier
}

// NewCoreOIDCIdentityVerifier constructs the default go-oidc-backed verifier adapter.
func NewCoreOIDCIdentityVerifier(verifier *oidc.IDTokenVerifier) CoreOIDCIdentityVerifier {
	return CoreOIDCIdentityVerifier{verifier: verifier}
}

// Verify validates the raw ID token and exposes its claims for normalization.
func (v CoreOIDCIdentityVerifier) Verify(ctx context.Context, rawIDToken string) (OIDCVerifiedClaims, error) {
	if v.verifier == nil {
		return nil, fmt.Errorf("service: oidc verifier is required")
	}
	return v.verifier.Verify(ctx, strings.TrimSpace(rawIDToken))
}

type oidcIdentityClaims struct {
	Subject           string `json:"sub"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
}

func normalizedOIDCIdentity(ctx context.Context, verifier OIDCIdentityVerifier, provider string, rawIDToken string) (AuthIdentity, error) {
	if verifier == nil {
		return AuthIdentity{}, fmt.Errorf("service: oidc verifier is required")
	}
	token, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return AuthIdentity{}, err
	}
	var claims oidcIdentityClaims
	if err := token.Claims(&claims); err != nil {
		return AuthIdentity{}, err
	}
	login := strings.TrimSpace(claims.PreferredUsername)
	if login == "" {
		login = strings.TrimSpace(claims.Name)
	}
	if strings.TrimSpace(provider) == "" || strings.TrimSpace(claims.Subject) == "" || login == "" {
		return AuthIdentity{}, fmt.Errorf("service: oidc identity claims are incomplete")
	}
	return AuthIdentity{
		Provider: strings.TrimSpace(provider),
		Subject:  strings.TrimSpace(claims.Subject),
		Login:    login,
		Email:    strings.TrimSpace(claims.Email),
	}, nil
}
